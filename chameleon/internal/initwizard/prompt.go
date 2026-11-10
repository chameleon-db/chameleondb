// Package initwizard holds the interactive part of `chameleon init` and of the
// first `migrate --apply`. It depends on the standard library and yaml only, so
// that it can be tested without a database, a terminal or the Rust core.
package initwizard

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Prompter asks numbered and yes/no questions. End of input answers every
// question with its default, so a closed stdin never blocks.
type Prompter struct {
	In  *bufio.Reader
	Out io.Writer
}

// NewPrompter wraps in and out.
func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{In: bufio.NewReader(in), Out: out}
}

func (p *Prompter) readLine() (string, bool) {
	line, err := p.In.ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.TrimSpace(line), true
}

// Choose prints options (1-based) and returns the 0-based index chosen. Enter
// or end of input select def (0-based). An invalid answer asks again, up to
// three times, and then falls back to def.
func (p *Prompter) Choose(question string, options []string, def int) int {
	fmt.Fprintln(p.Out, question)
	for i, o := range options {
		fmt.Fprintf(p.Out, "  %d) %s\n", i+1, o)
	}

	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintf(p.Out, "Choice [%d]: ", def+1)
		line, ok := p.readLine()
		if !ok || line == "" {
			fmt.Fprintln(p.Out)
			return def
		}
		if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(options) {
			return n - 1
		}
		fmt.Fprintf(p.Out, "Please enter a number between 1 and %d.\n", len(options))
	}

	return def
}

// Confirm asks a yes/no question. Enter and end of input select def.
func (p *Prompter) Confirm(question string, def bool) bool {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}

	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintf(p.Out, "%s %s: ", question, hint)
		line, ok := p.readLine()
		if !ok || line == "" {
			fmt.Fprintln(p.Out)
			return def
		}
		switch strings.ToLower(line) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		fmt.Fprintln(p.Out, "Please answer y or n.")
	}

	return def
}

// Ask reads a free-text answer. Enter and end of input return def.
func (p *Prompter) Ask(question, def string) string {
	if def != "" {
		fmt.Fprintf(p.Out, "%s [%s]: ", question, def)
	} else {
		fmt.Fprintf(p.Out, "%s: ", question)
	}
	line, ok := p.readLine()
	if !ok || line == "" {
		fmt.Fprintln(p.Out)
		return def
	}
	return line
}
