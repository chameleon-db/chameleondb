package cliout

import "regexp"

const redacted = "***"

var (
	// scheme://user:password@host: the password may hold any character that is
	// not a separator of the URL.
	urlPassword = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://[^\s:/@"']*:)[^\s/@"']+(@)`)
	// password=value in a key/value connection string, and PGPASSWORD=value.
	kvPassword = regexp.MustCompile(`(?i)(\b(?:pg)?password\s*=\s*)('[^']*'|[^\s"']+)`)
)

// Redact removes passwords from connection strings found in s. It is applied to
// everything the CLI prints in JSON and to the error messages, because driver
// errors and URLs typed by the user can carry credentials.
func Redact(s string) string {
	s = urlPassword.ReplaceAllString(s, "${1}"+redacted+"${2}")
	s = kvPassword.ReplaceAllString(s, "${1}"+redacted)
	return s
}
