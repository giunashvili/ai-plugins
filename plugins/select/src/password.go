package main

import (
	"errors"
	"strings"
)

func storedPassword(c Connection) (string, error) {
	if c.Password == nil {
		return "", errors.New("no password saved in profiles.json; enter it in Select setup (older OS-stored passwords must be saved again)")
	}
	if strings.ContainsRune(*c.Password, 0) {
		return "", errors.New("password cannot contain a NUL character")
	}
	return *c.Password, nil
}
