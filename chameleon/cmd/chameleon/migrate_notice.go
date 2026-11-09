package main

import "strings"

// readonlyNotice explains, at the end of a successful `migrate --apply`, that
// the project is in readonly mode and how to leave it. It returns an empty
// string for any other mode.
func readonlyNotice(mode string) string {
	if mode != "readonly" {
		return ""
	}

	var b strings.Builder
	b.WriteString("Paranoid mode: readonly\n")
	b.WriteString("  The schema is now locked: another 'migrate --apply' will be refused.\n")
	b.WriteString("  To change the schema later:\n")
	b.WriteString("    1. chameleon config auth set-password\n")
	b.WriteString("    2. chameleon config set mode=standard\n")
	return b.String()
}
