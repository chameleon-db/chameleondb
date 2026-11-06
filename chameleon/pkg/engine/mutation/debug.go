package mutation

import (
	"bytes"
	"sort"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// formatDebugParams renders statement parameters through the engine debug
// context. It remains a small test helper for the legacy formatter contract.
func formatDebugParams(labels []string, values []interface{}) string {
	var out bytes.Buffer
	debug := &engine.DebugContext{Level: engine.DebugSQL, Writer: &out}
	debug.LogNamedParams(labels, values)
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
	return effectiveDebugContext(nil, override).Level
}

func effectiveDebugContext(base *engine.DebugContext, override *engine.DebugLevel) *engine.DebugContext {
	if base == nil {
		base = engine.DebugContextFromEnv()
	}
	if override == nil {
		return base
	}

	context := *base
	context.Level = *override
	return &context
}
