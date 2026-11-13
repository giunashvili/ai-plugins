package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
)

var errReadOnly = errors.New("only a single read-only SELECT, SHOW, DESCRIBE or EXPLAIN query is allowed")

func readStatement(stmt ast.StmtNode) bool {
	switch s := stmt.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt, *ast.ShowStmt:
		return true
	case *ast.ExplainStmt:
		return readStatement(s.Stmt)
	default:
		return false
	}
}

type readVisitor struct{ denied bool }

func (v *readVisitor) Enter(node ast.Node) (ast.Node, bool) {
	switch n := node.(type) {
	case *ast.SelectStmt:
		if n.SelectIntoOpt != nil || n.LockInfo != nil {
			v.denied = true
		}
	case *ast.VariableExpr:
		if n.Value != nil {
			v.denied = true
		}
	case *ast.FuncCallExpr:
		if n.Schema.O != "" {
			v.denied = true
		}
		switch strings.ToUpper(n.FnName.O) {
		case "GET_LOCK", "RELEASE_LOCK", "RELEASE_ALL_LOCKS", "LOAD_FILE", "SLEEP", "BENCHMARK", "NEXTVAL", "SETVAL":
			v.denied = true
		}
	}
	return node, v.denied
}
func (v *readVisitor) Leave(node ast.Node) (ast.Node, bool) { return node, !v.denied }
func validateSQL(query string) error {
	if len(query) == 0 || len(query) > 1000000 || strings.Contains(query, "/*!") || strings.Contains(strings.ToUpper(query), "/*M!") {
		return errReadOnly
	}
	statements, _, err := parser.New().ParseSQL(query)
	if err != nil || len(statements) != 1 || !readStatement(statements[0]) {
		return errReadOnly
	}
	visitor := &readVisitor{}
	statements[0].Accept(visitor)
	if visitor.denied {
		return errReadOnly
	}
	return nil
}

type QueryResult struct {
	Rows      []map[string]any `json:"rows"`
	RowCount  int              `json:"rowCount"`
	Truncated bool             `json:"truncated"`
}

// Classify failures without returning server messages, DSNs or credentials.
func connectionError(err error, stage string) error {
	message := "check network access and database settings"
	var server *mysql.MySQLError
	var dns *net.DNSError
	var unknownCA x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var invalidCert x509.CertificateInvalidError
	var network net.Error
	switch {
	case errors.As(err, &server):
		switch server.Number {
		case 1045, 1698:
			message = "authentication rejected; check the saved username/password, allowed client hosts and whether the account requires TLS"
		case 1044:
			message = "the database account does not have access to this database"
		case 1049:
			message = "the configured database does not exist"
		case 1130:
			message = "this client host is not allowed to connect"
		case 3159:
			message = "the server requires a secure connection; enable TLS for this environment"
		case 1040, 1203:
			message = "the server or account connection limit has been reached"
		default:
			message = "the server rejected the request; check database permissions and server compatibility"
		}
		return fmt.Errorf("%s: MySQL error %d; %s", stage, server.Number, message)
	case errors.Is(err, context.Canceled):
		message = "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		message = "connection timed out; check network/VPN access and firewall rules"
	case errors.As(err, &dns):
		message = "database hostname could not be resolved; check the hostname, DNS and VPN"
	case errors.As(err, &unknownCA):
		message = "TLS certificate authority is not trusted; configure the correct CA certificate"
	case errors.As(err, &hostname):
		message = "TLS certificate does not match the configured database hostname"
	case errors.As(err, &invalidCert):
		message = "TLS certificate is invalid or expired; check the certificate and system clock"
	case errors.Is(err, mysql.ErrNoTLS):
		message = "TLS is enabled but this server does not support TLS"
	case errors.Is(err, syscall.ECONNREFUSED):
		message = "connection refused; check the database host, port and firewall"
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
		message = "network access was denied by this app's execution environment"
	case errors.As(err, &network) && network.Timeout():
		message = "connection timed out; check network/VPN access and firewall rules"
	case errors.Is(err, mysql.ErrUnknownPlugin), errors.Is(err, mysql.ErrOldPassword), errors.Is(err, mysql.ErrCleartextPassword):
		message = "the database account requires an unsupported or insecure authentication method"
	}
	return fmt.Errorf("%s: %s", stage, message)
}

func runQuery(ctx context.Context, c Connection, password, query string) (QueryResult, error) {
	result := QueryResult{Rows: []map[string]any{}}
	if err := validateSQL(query); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	cfg.DBName = c.Database
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 45 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	cfg.MultiStatements = false
	cfg.AllowAllFiles = false
	cfg.AllowCleartextPasswords = false
	cfg.AllowFallbackToPlaintext = false
	cfg.Logger = log.New(io.Discard, "", 0)
	if c.TLS.Enabled {
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if c.TLS.CAFile != "" {
			pem, err := os.ReadFile(c.TLS.CAFile)
			if err != nil || !roots.AppendCertsFromPEM(pem) {
				return result, errors.New("could not load the TLS CA certificate")
			}
		}
		cfg.TLS = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: c.Host}
	}
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return result, errors.New("invalid database connection settings")
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = db.PingContext(ctx); err != nil {
		return result, connectionError(err, "could not connect")
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, connectionError(err, "could not start a read-only transaction")
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return result, errors.New("query failed; check SQL and read permissions")
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return result, errors.New("could not read query columns")
	}
	for _, name := range names {
		if len(name) > 4096 {
			return result, errors.New("query column name is too long")
		}
	}
	size := 0
	for rows.Next() {
		if len(result.Rows) >= 1000 {
			result.Truncated = true
			break
		}
		values := make([]any, len(names))
		ptrs := make([]any, len(names))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return result, errors.New("could not read query rows")
		}
		row := make(map[string]any, len(names))
		for i, name := range names {
			if b, ok := values[i].([]byte); ok {
				row[name] = string(b)
			} else {
				row[name] = values[i]
			}
		}
		b, _ := json.Marshal(row)
		size += len(b)
		if size > 4<<20 {
			result.Truncated = true
			break
		}
		result.Rows = append(result.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return result, errors.New("query failed or timed out")
	}
	result.RowCount = len(result.Rows)
	return result, nil
}
