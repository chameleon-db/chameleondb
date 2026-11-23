package main

import (
	"fmt"
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/internal/cliout"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Machine-readable output.
//
// With --json a command prints exactly one JSON document on stdout (the
// envelope of internal/cliout). Everything meant for people (progress lines,
// warnings, the SQL preview) keeps being printed by the same code, but goes to
// stderr. Commands describe their result with setResult; failures travel as
// errors, ideally *cliout.Error, which carries the code that decides the exit
// status. The exit status is the same with and without --json.

const jsonAnnotation = "chameleon/json"

var (
	// jsonOutput is the value of --json.
	jsonOutput bool
	// jsonMode is true once stdout was redirected to stderr for a command that
	// supports --json.
	jsonMode bool
	// jsonStdout is the real stdout while jsonMode is on.
	jsonStdout *os.File
	// jsonResult is the "data" of the envelope.
	jsonResult any
	// jsonCommand is the command being run, for the envelope.
	jsonCommand string
)

// supportJSON marks a command as able to print the envelope.
func supportJSON(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[jsonAnnotation] = "true"
}

// setResult records the data of the envelope. It is ignored without --json.
func setResult(data any) {
	jsonResult = data
}

// noInput reports whether questions are off: the command's --no-input flag, or
// --json, because nobody reads a prompt when the output is a document.
func noInput(flag bool) bool {
	return flag || jsonOutput
}

// prepareOutput runs before every command (PersistentPreRunE of the root).
func prepareOutput(cmd *cobra.Command, _ []string) error {
	jsonCommand = cmd.CommandPath()
	if !jsonOutput {
		return nil
	}
	if cmd.Annotations[jsonAnnotation] != "true" {
		return cliout.New(cliout.CodeUsage, fmt.Sprintf("--json is not supported by %q", cmd.CommandPath())).
			WithHint("--json works with: validate, status, verify, migrate, introspect")
	}
	enableJSONMode()
	return nil
}

// enableJSONMode sends everything written to stdout to stderr, keeping the
// real stdout for the envelope.
func enableJSONMode() {
	jsonMode = true
	jsonStdout = os.Stdout
	os.Stdout = os.Stderr
	color.Output = os.Stderr
	color.NoColor = true
}

// Execute runs the CLI and exits with the status of the result.
func Execute() {
	// A flag error happens before any hook runs, so --json is looked for in the
	// arguments too: a script that asked for JSON gets JSON for every failure.
	for _, a := range os.Args[1:] {
		if a == "--" {
			break
		}
		if a == "--json" || a == "--json=true" {
			jsonOutput = true
		}
	}
	os.Exit(finish(rootCmd.Execute()))
}

// finish prints the outcome and returns the exit status.
func finish(err error) int {
	status := cliout.ExitCodeOf(err)

	if jsonOutput {
		command := jsonCommand
		if command == "" {
			command = rootCmd.CommandPath()
		}
		var env cliout.Envelope
		if err != nil {
			env = cliout.Failure(command, err, jsonResult)
		} else {
			env = cliout.Success(command, jsonResult)
		}
		out := jsonStdout
		if out == nil {
			out = os.Stdout
		}
		if werr := cliout.Write(out, env); werr != nil {
			fmt.Fprintf(os.Stderr, "Error: could not write JSON output: %v\n", werr)
			if status == cliout.ExitOK {
				status = cliout.ExitInternal
			}
		}
		return status
	}

	if err != nil && !cliout.IsSilent(err) {
		errorColor.Fprintf(os.Stderr, "Error: %v\n", cliout.Redact(err.Error()))
	}
	return status
}
