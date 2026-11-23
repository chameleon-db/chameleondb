// Package cliout defines what the command line tells scripts, CI jobs and
// coding agents: one JSON envelope for --json, stable error codes and the exit
// code of each class of failure. It has no dependency on the commands, so the
// contract can be tested on its own.
package cliout

// SchemaVersion is the version of the JSON envelope. It changes only when a
// field of the envelope is removed or changes meaning; adding fields does not
// change it.
const SchemaVersion = 1

// Code is a stable, machine-readable identifier of a class of failure.
type Code string

const (
	// CodeInternal is any failure that has no more specific class.
	CodeInternal Code = "internal_error"
	// CodeUsage is a wrong flag, argument or command.
	CodeUsage Code = "usage_error"
	// CodeSchemaInvalid is a schema that does not parse or type-check.
	CodeSchemaInvalid Code = "schema_invalid"
	// CodeModeBlocked is an operation refused by the paranoid mode.
	CodeModeBlocked Code = "mode_blocked"
	// CodeIntegrityFailed is a vault whose files do not match their hashes, or
	// whose mode.json was edited outside the CLI.
	CodeIntegrityFailed Code = "integrity_failed"
	// CodeConnectionFailed is a database that could not be reached.
	CodeConnectionFailed Code = "connection_failed"
	// CodeNotInitialized is a directory that is not a ChameleonDB project (no
	// configuration or no vault).
	CodeNotInitialized Code = "not_initialized"
	// CodeApplyFailed is a migration that the database rejected.
	CodeApplyFailed Code = "apply_failed"
)

// Exit codes. Zero is success; every failure is non-zero.
const (
	ExitOK             = 0
	ExitInternal       = 1
	ExitUsage          = 2
	ExitSchemaInvalid  = 3
	ExitModeBlocked    = 4
	ExitIntegrity      = 5
	ExitConnection     = 6
	ExitNotInitialized = 7
	ExitApplyFailed    = 8
)

// Entry is one row of the contract: a code and its exit code.
type Entry struct {
	Code        Code
	Exit        int
	Description string
}

// Table lists every code with its exit code, in exit-code order. The docs and
// the tests are generated from it, so a code cannot exist without being listed.
var Table = []Entry{
	{CodeInternal, ExitInternal, "Unexpected failure with no more specific class"},
	{CodeUsage, ExitUsage, "Wrong flag, argument or command"},
	{CodeSchemaInvalid, ExitSchemaInvalid, "The schema does not parse or type-check"},
	{CodeModeBlocked, ExitModeBlocked, "The operation is refused by the paranoid mode"},
	{CodeIntegrityFailed, ExitIntegrity, "Vault integrity check failed, or mode.json was edited outside the CLI"},
	{CodeConnectionFailed, ExitConnection, "The database could not be reached"},
	{CodeNotInitialized, ExitNotInitialized, "Not a ChameleonDB project (no configuration or no vault)"},
	{CodeApplyFailed, ExitApplyFailed, "The database rejected the migration"},
}

// ExitCode returns the process exit code for the code.
func (c Code) ExitCode() int {
	for _, e := range Table {
		if e.Code == c {
			return e.Exit
		}
	}
	return ExitInternal
}
