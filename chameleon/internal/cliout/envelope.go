package cliout

import (
	"bytes"
	"encoding/json"
	"io"
)

// Envelope is the one JSON document a command prints with --json.
type Envelope struct {
	SchemaVersion int      `json:"schema_version"`
	Command       string   `json:"command"`
	OK            bool     `json:"ok"`
	Data          any      `json:"data"`
	Errors        []Detail `json:"errors"`
}

// Success builds the envelope of a command that worked.
func Success(command string, data any) Envelope {
	return Envelope{SchemaVersion: SchemaVersion, Command: command, OK: true, Data: data, Errors: []Detail{}}
}

// Failure builds the envelope of a command that failed. data may be nil, or
// hold what the command found before it failed (for example the versions that
// failed verification).
func Failure(command string, err error, data any) Envelope {
	details := DetailsOf(err)
	if details == nil {
		details = []Detail{}
	}
	return Envelope{SchemaVersion: SchemaVersion, Command: command, OK: false, Data: data, Errors: details}
}

// Marshal renders the envelope: indented, with "data" an object and "errors" an
// array even when empty, and with credentials removed from every string.
func Marshal(env Envelope) ([]byte, error) {
	if env.Data == nil {
		env.Data = struct{}{}
	}
	if env.Errors == nil {
		env.Errors = []Detail{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return nil, err
	}
	return []byte(Redact(buf.String())), nil
}

// Write prints the envelope followed by a newline.
func Write(w io.Writer, env Envelope) error {
	b, err := Marshal(env)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
