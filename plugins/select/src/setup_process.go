package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gofrs/flock"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type setupSession struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}
type windowControl struct {
	ready atomic.Bool
	mu    sync.Mutex
	focus func()
}

var setupLaunchMu sync.Mutex

func sessionPath() string { return filepath.Join(configDirectory(false), ".setup-session.json") }
func focusSetup() bool {
	b, err := os.ReadFile(sessionPath())
	if err != nil || len(b) > 1024 {
		return false
	}
	var s setupSession
	if decodeJSON(b, &s) != nil || s.Port < 1 || s.Port > 65535 || len(s.Token) != 64 {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(s.Port)), time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if json.NewEncoder(conn).Encode(s.Token) != nil {
		return false
	}
	var ready bool
	return json.NewDecoder(io.LimitReader(conn, 64)).Decode(&ready) == nil && ready
}
func openSetup(ctx context.Context) error {
	setupLaunchMu.Lock()
	defer setupLaunchMu.Unlock()
	if focusSetup() {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate the settings executable")
	}
	child := exec.Command(exe, "setup", "--from-mcp")
	child.Stderr = io.Discard
	child.Stdin = nil
	detachSetup(child)
	stdout, err := child.StdoutPipe()
	if err != nil {
		return errors.New("could not open settings")
	}
	if err = child.Start(); err != nil {
		return errors.New("could not start settings; check executable permissions and system WebView dependencies")
	}
	ready := make(chan bool, 1)
	go func() {
		// Reap only after reading: Wait closes StdoutPipe, which can otherwise
		// race a short-lived child acknowledging an existing settings window.
		defer child.Wait()
		scan := bufio.NewScanner(stdout)
		scan.Buffer(make([]byte, 256), 4096)
		for scan.Scan() {
			if scan.Text() == `{"ready":true}` {
				ready <- true
				io.Copy(io.Discard, stdout)
				return
			}
		}
		ready <- false
	}()
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	select {
	case ok := <-ready:
		if ok {
			return nil
		}
	case <-ctx.Done():
		child.Process.Kill()
		return errors.New("opening settings was cancelled")
	case <-timer.C:
		child.Process.Kill()
	}
	return errors.New("settings could not open; check your desktop session, WebView2 on Windows, or GTK/WebKit on Linux")
}
func setupProcess(ctx context.Context, fromMCP bool) error {
	dir := configDirectory(false)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return errors.New("could not open the configuration directory")
	}
	lock := flock.New(filepath.Join(dir, ".setup.lock"))
	ok, err := lock.TryLock()
	if err != nil {
		return errors.New("could not lock settings")
	}
	if !ok {
		deadline := time.Now().Add(18 * time.Second)
		for time.Now().Before(deadline) {
			if focusSetup() {
				if fromMCP {
					io.WriteString(os.Stdout, `{"ready":true}`+"\n")
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		return errors.New("another settings process is starting; try again")
	}
	defer lock.Unlock()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errors.New("could not start local window control")
	}
	defer listener.Close()
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return errors.New("could not create window control token")
	}
	session := setupSession{Port: listener.Addr().(*net.TCPAddr).Port, Token: hex.EncodeToString(random)}
	b, _ := json.Marshal(session)
	f, err := os.CreateTemp(dir, ".setup-session-*")
	if err != nil {
		return err
	}
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(f.Name(), sessionPath())
	}
	os.Remove(f.Name())
	if err != nil {
		return errors.New("could not save window session")
	}
	defer os.Remove(sessionPath())
	control := &windowControl{}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(2 * time.Second))
				var token string
				if json.NewDecoder(io.LimitReader(conn, 256)).Decode(&token) != nil || len(token) != len(session.Token) || subtle.ConstantTimeCompare([]byte(token), []byte(session.Token)) != 1 {
					return
				}
				control.mu.Lock()
				ready := control.ready.Load()
				if ready && control.focus != nil {
					control.focus()
				}
				control.mu.Unlock()
				json.NewEncoder(conn).Encode(ready)
			}()
		}
	}()
	return runSetupWindow(ctx, control, func() {
		control.ready.Store(true)
		if fromMCP {
			io.WriteString(os.Stdout, `{"ready":true}`+"\n")
		}
	})
}
