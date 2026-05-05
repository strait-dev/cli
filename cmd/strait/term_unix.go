//go:build !windows

package main

import (
	"syscall"

	"golang.org/x/term"
)

func stdinIsTerminal() bool {
	return term.IsTerminal(syscall.Stdin)
}

func readStdinPassword() ([]byte, error) {
	return term.ReadPassword(syscall.Stdin)
}
