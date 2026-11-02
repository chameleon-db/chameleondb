package engine

import (
	"fmt"
	"strings"
)

// maxIdentLen is the maximum identifier length accepted by PostgreSQL.
const maxIdentLen = 63

// QuoteIdent validates a name and renders it as a quoted PostgreSQL
// identifier. It enforces the same rules as quote_ident in the Rust core
// (chameleon-core/src/sql/naming.rs): the name must match
// [A-Za-z_][A-Za-z0-9_]{0,62}, and it is lower-cased, which is how PostgreSQL
// treats the unquoted names emitted by the migration generator.
//
// It is the only place the Go runtime turns a name into SQL text.
func QuoteIdent(name string) (string, error) {
	if !validIdent(name) {
		return "", fmt.Errorf("invalid identifier")
	}
	return `"` + strings.ToLower(name) + `"`, nil
}

func validIdent(name string) bool {
	if name == "" || len(name) > maxIdentLen {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
