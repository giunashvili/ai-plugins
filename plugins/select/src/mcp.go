package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"strconv"
)

const version = "0.6.0"

type QueryInput struct {
	Project     string `json:"project" jsonschema:"Explicit configured project identifier"`
	Environment string `json:"environment" jsonschema:"Explicit configured environment identifier"`
	SQL         string `json:"sql" jsonschema:"A single read-only MySQL query"`
}
type DesktopQuery struct {
	SQL string `json:"sql"`
}

func textResult(value any) *mcp.CallToolResult {
	b, _ := json.Marshal(value)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: message}}}
}
func hint(read, world bool) *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{ReadOnlyHint: read, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &world}
}
func runMCP(ctx context.Context, desktop bool) error {
	server := mcp.NewServer(&mcp.Implementation{Name: "select", Version: version}, nil)
	if desktop {
		c, p, err := desktopConnection()
		if err != nil {
			return err
		}
		mcp.AddTool(server, &mcp.Tool{Name: "mysql_query", Title: "Query the configured database", Description: "Run a single read-only MySQL query on this extension's configured connection.", Annotations: hint(true, true)}, func(ctx context.Context, _ *mcp.CallToolRequest, input DesktopQuery) (*mcp.CallToolResult, any, error) {
			r, err := runQuery(ctx, c, p, input.SQL)
			if err != nil {
				return toolError(err.Error()), nil, nil
			}
			return textResult(r), nil, nil
		})
	} else {
		mcp.AddTool(server, &mcp.Tool{Name: "db_status", Title: "List database projects", Description: "List projects and environments with configured status, color and emoji, without credentials. Configured means saved connection details and password, not verified connectivity.", Annotations: hint(true, false)}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			r, err := statusData()
			if err != nil {
				return toolError("Could not read database settings."), nil, nil
			}
			return textResult(r), nil, nil
		})
		mcp.AddTool(server, &mcp.Tool{Name: "db_setup", Title: "Open Select settings", Description: "Open the native settings window. The user enters credentials directly in that window; never request passwords in chat.", Annotations: hint(false, false)}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			if err := openSetup(ctx); err != nil {
				return toolError(err.Error()), nil, nil
			}
			return textResult(map[string]bool{"opened": true}), nil, nil
		})
		mcp.AddTool(server, &mcp.Tool{Name: "mysql_query", Title: "Query a project database", Description: "Run a single read-only query for an explicitly selected project and environment.", Annotations: hint(true, true)}, func(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
			// Reject unsafe SQL before loading connection credentials.
			if err := validateSQL(input.SQL); err != nil {
				return toolError(err.Error()), nil, nil
			}
			c, err := getConnection(input.Project, input.Environment)
			if err != nil {
				return toolError("Project/environment is not configured. Open Select setup."), nil, nil
			}
			password, err := storedPassword(c)
			if err != nil {
				return toolError(err.Error()), nil, nil
			}
			r, err := runQuery(ctx, c, password, input.SQL)
			if err != nil {
				return toolError(err.Error()), nil, nil
			}
			return textResult(r), nil, nil
		})
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}
func desktopConnection() (Connection, string, error) {
	if !validName(os.Getenv("DB_MCPB_PROJECT")) || !validName(os.Getenv("DB_MCPB_ENVIRONMENT")) {
		return Connection{}, "", errors.New("set a project and environment in extension settings")
	}
	tlsValue := os.Getenv("DB_MCPB_TLS")
	if tlsValue != "true" && tlsValue != "false" {
		return Connection{}, "", errors.New("set TLS in extension settings")
	}
	port := 3306
	var err error
	if value := os.Getenv("DB_MCPB_PORT"); value != "" {
		port, err = strconv.Atoi(value)
		if err != nil {
			return Connection{}, "", errors.New("invalid extension port")
		}
	}
	c, err := validateConnection(Connection{Host: os.Getenv("DB_MCPB_HOST"), Port: port, Database: os.Getenv("DB_MCPB_DATABASE"), User: os.Getenv("DB_MCPB_USER"), TLS: TLSConfig{Enabled: tlsValue == "true", CAFile: os.Getenv("DB_MCPB_CA_FILE")}})
	p, ok := os.LookupEnv("DB_MCPB_PASSWORD")
	if !ok {
		return c, "", errors.New("set the password in extension settings")
	}
	return c, p, err
}
