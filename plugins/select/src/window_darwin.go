//go:build darwin && !headless

package main

// #cgo LDFLAGS: -framework UniformTypeIdentifiers
import "C"

// Direct Go builds need UniformTypeIdentifiers for Wails' UTType file dialogs.
