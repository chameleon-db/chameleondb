package mutation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// maskedValue replaces the value of a sensitive column in debug output.
const maskedValue = "***"

// sensitiveFragments are lowercase fragments of column names whose values are
// never printed by the mutation debug output.
var sensitiveFragments = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private_key", "credential", "hash", "salt",
}

// isSensitiveField reports whether a column holds a value that should not be
// written to debug output.
func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, fragment := range sensitiveFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// formatDebugParams renders statement parameters like the query builder does
// ("$1 = value"), masking sensitive columns. labels are the column names of
// the parameters in order. When they do not line up with the values, every
// value is masked: showing nothing is safer than showing the wrong thing.
func formatDebugParams(labels []string, values []interface{}) string {
	if len(values) == 0 {
		return ""
	}

	aligned := len(labels) == len(values)

	var out strings.Builder
	out.WriteString("[PARAMS]\n")
	for i, value := range values {
		if !aligned || isSensitiveField(labels[i]) {
			fmt.Fprintf(&out, "$%d = %s\n", i+1, maskedValue)
			continue
		}
		fmt.Fprintf(&out, "$%d = %v\n", i+1, value)
	}
	out.WriteString("\n")
	return out.String()
}

// sortedKeys returns the keys of a map in the order the SQL builders use.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// filterLabels returns the column of each filter, in the order the SQL
// builders bind them. A filter key is "field" or "field:op".
func filterLabels(filters map[string]interface{}) []string {
	keys := sortedKeys(filters)
	labels := make([]string, 0, len(keys))
	for _, key := range keys {
		field, _, _ := strings.Cut(key, ":")
		labels = append(labels, field)
	}
	return labels
}

// effectiveDebugLevel is the level set with .Debug() on the builder, or the
// one from CHAMELEON_DEBUG when the builder has no override.
func effectiveDebugLevel(override *engine.DebugLevel) engine.DebugLevel {
	if override != nil {
		return *override
	}
	return engine.DebugContextFromEnv().Level
}
