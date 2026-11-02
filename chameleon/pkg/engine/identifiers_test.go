package engine

import (
	"strings"
	"testing"
)

func TestQuoteIdentAcceptsPlainNames(t *testing.T) {
	cases := map[string]string{
		"users":                 `"users"`,
		"_a1":                   `"_a1"`,
		"createdAt":             `"createdat"`,
		strings.Repeat("a", 63): `"` + strings.Repeat("a", 63) + `"`,
	}
	for name, want := range cases {
		got, err := QuoteIdent(name)
		if err != nil || got != want {
			t.Errorf("QuoteIdent(%q) = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestQuoteIdentRejectsOtherNames(t *testing.T) {
	for _, name := range []string{
		"", "1a", "a b", "a-b", "a.b", `a"b`, "a'b", "a;b", "a--b", "a/*b",
		`a\b`, "a\x00b", "éa", "aé", strings.Repeat("a", 64),
	} {
		if got, err := QuoteIdent(name); err == nil {
			t.Errorf("QuoteIdent(%q) accepted, got %q", name, got)
		}
	}
}
