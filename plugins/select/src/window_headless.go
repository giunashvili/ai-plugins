//go:build headless

package main

import (
	"context"
	"errors"
)

func runSetupWindow(context.Context, *windowControl, func()) error {
	return errors.New("this development-only headless build has no settings window; build without the headless tag")
}
