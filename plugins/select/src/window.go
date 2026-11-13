//go:build !headless

package main

import (
	"context"
	"embed"
	"errors"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed ui/*
var assets embed.FS

func setupAssets(executable string) (fs.FS, bool, error) {
	ui, err := fs.Sub(assets, "ui")
	if err != nil {
		return nil, false, errors.New("embedded settings files unavailable")
	}
	directory := os.Getenv("SELECT_UI_DIR")
	if directory == "" && filepath.Base(filepath.Dir(filepath.Dir(executable))) == "build" {
		candidate := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(executable))), "src", "ui")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			directory = candidate
		}
	}
	if directory == "" {
		return ui, false, nil
	}
	if !filepath.IsAbs(directory) {
		return nil, false, errors.New("development UI directory must be an absolute path")
	}
	for _, name := range []string{"index.html", "style.css", "app.js"} {
		if info, err := os.Stat(filepath.Join(directory, name)); err != nil || !info.Mode().IsRegular() {
			return nil, false, errors.New("development UI files unavailable; check SELECT_UI_DIR or the repository location")
		}
	}
	return os.DirFS(directory), true, nil
}

func noAssetCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type quietLogger struct{}

func (quietLogger) Print(string)   {}
func (quietLogger) Trace(string)   {}
func (quietLogger) Debug(string)   {}
func (quietLogger) Info(string)    {}
func (quietLogger) Warning(string) {}
func (quietLogger) Error(string)   {}
func (quietLogger) Fatal(string)   {}
func runSetupWindow(ctx context.Context, control *windowControl, ready func()) error {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return errors.New("settings require a graphical desktop session")
	}
	executable, err := os.Executable()
	if err != nil {
		return errors.New("could not locate the settings executable")
	}
	ui, development, err := setupAssets(executable)
	if err != nil {
		return err
	}
	err = wails.Run(&options.App{Title: "Select · Settings", Width: 960, Height: 760, MinWidth: 680, MinHeight: 600, AssetServer: &assetserver.Options{Assets: ui, Middleware: noAssetCache}, Bind: []interface{}{&App{development: development}}, Logger: quietLogger{},
		OnStartup: func(wctx context.Context) {
			control.mu.Lock()
			control.focus = func() { wruntime.WindowShow(wctx); wruntime.WindowUnminimise(wctx) }
			control.mu.Unlock()
			go func() { <-ctx.Done(); wruntime.Quit(wctx) }()
		},
		OnDomReady: func(context.Context) { ready() }, OnShutdown: func(context.Context) { control.ready.Store(false) },
	})
	if err != nil {
		return errors.New("could not open settings; verify the desktop WebView runtime")
	}
	return nil
}
