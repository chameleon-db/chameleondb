package cliout

import (
	"errors"
	"fmt"
	"strings"
)

// Detail is one entry of the "errors" array of the envelope.
type Detail struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// Error is an error that carries its class. Commands return it where they know
// the class; anything else is reported as CodeInternal.
type Error struct {
	Code    Code
	Message string
	Hint    string
	// Details, when set, replace the single entry derived from Message in the
	// envelope (a schema with three problems reports three entries).
	Details []Detail
	// Silent means the command already printed the explanation for people, so
	// the text mode must not print "Error: ..." again.
	Silent bool
	Err    error
}

// New returns an Error with a fixed message.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Errorf formats a message; %w wraps the cause as in fmt.Errorf.
func Errorf(code Code, format string, args ...any) *Error {
	err := fmt.Errorf(format, args...)
	return &Error{Code: code, Message: err.Error(), Err: errors.Unwrap(err)}
}

// Wrap gives an existing error a class, keeping its message.
func Wrap(code Code, err error) *Error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return existing
	}
	return &Error{Code: code, Message: err.Error(), Err: err}
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Err }

// WithHint adds the next step for the reader.
func (e *Error) WithHint(hint string) *Error {
	e.Hint = hint
	return e
}

// WithDetails sets the entries of the envelope.
func (e *Error) WithDetails(details ...Detail) *Error {
	e.Details = details
	return e
}

// AsSilent marks the error as already explained in the human output.
func (e *Error) AsSilent() *Error {
	e.Silent = true
	return e
}

// CodeOf returns the class of err: the code of the first *Error in the chain,
// CodeUsage for the messages cobra produces for bad flags and arguments, and
// CodeInternal otherwise. It returns "" for nil.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	if IsUsageMessage(err.Error()) {
		return CodeUsage
	}
	return CodeInternal
}

// ExitCodeOf returns the process exit code for err (0 for nil).
func ExitCodeOf(err error) int {
	if err == nil {
		return ExitOK
	}
	return CodeOf(err).ExitCode()
}

// IsSilent reports whether the human output already explained err.
func IsSilent(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Silent
}

// DetailsOf returns the entries of the envelope for err.
func DetailsOf(err error) []Detail {
	if err == nil {
		return nil
	}
	code := CodeOf(err)
	var e *Error
	if errors.As(err, &e) {
		if len(e.Details) > 0 {
			out := make([]Detail, len(e.Details))
			for i, d := range e.Details {
				if d.Code == "" {
					d.Code = code
				}
				out[i] = d
			}
			return out
		}
		return []Detail{{Code: code, Message: e.Message, Hint: e.Hint}}
	}
	return []Detail{{Code: code, Message: err.Error()}}
}

// usagePrefixes are the beginnings of the messages cobra and pflag return for
// bad flags and arguments. They are the only way to recognise those errors:
// cobra returns plain errors for them.
var usagePrefixes = []string{
	"unknown command",
	"unknown flag",
	"unknown shorthand flag",
	"accepts ",
	"requires at least",
	"required flag",
	"flag needs an argument",
	"bad flag syntax",
	"invalid argument",
}

// IsUsageMessage reports whether msg looks like a cobra usage error.
func IsUsageMessage(msg string) bool {
	for _, p := range usagePrefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	return false
}
