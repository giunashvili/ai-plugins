package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	mode := "setup"
	args := os.Args[1:]
	if len(args) > 0 {
		mode = args[0]
		args = args[1:]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var err error
	switch mode {
	case "mcp", "desktop":
		if len(args) != 0 {
			err = fmt.Errorf("%s takes no arguments", mode)
		} else {
			err = runMCP(ctx, mode == "desktop")
		}
	case "setup":
		from := len(args) == 1 && args[0] == "--from-mcp"
		if len(args) > 0 && !from {
			err = fmt.Errorf("setup takes no arguments")
		} else {
			err = setupProcess(ctx, from)
		}
	case "version", "--version":
		fmt.Println(version)
	case "help", "--help", "-h":
		fmt.Println("Select\n\n  select mcp       Run the stdio MCP server\n  select setup     Open project/environment settings (default)\n  select desktop   Run a single connection from extension settings\n  select version   Print version\n\nNo Go or Node installation is required.")
	default:
		err = fmt.Errorf("unknown mode; use select help")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Select:", err)
		os.Exit(1)
	}
}
