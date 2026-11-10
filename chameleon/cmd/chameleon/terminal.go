package main

import (
	"os"

	"golang.org/x/term"
)

// interactiveTerminal reports whether questions may be asked: stdin and stdout
// are terminals, CI is not set, and --no-input was not passed.
func interactiveTerminal(noInput bool) bool {
	if noInput || os.Getenv("CI") != "" {
		return false
	}

	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
