package engine

import "fmt"

type IdentityMap struct {
	objects map[string]map[string]Row
}

func NewIdentityMap() *IdentityMap {
	return &IdentityMap{
		objects: make(map[string]map[string]Row),
	}
}

// Deduplicate processes rows and returns a deduplicated slice.
func (im *IdentityMap) Deduplicate(entity string, rows []Row) []Row {
	if len(rows) == 0 {
		return rows
	}

	if im.objects[entity] == nil {
		im.objects[entity] = make(map[string]Row)
	}

	result := make([]Row, 0, len(rows))

	for _, row := range rows {
		id := im.extractID(row)
		if id == "" {
			result = append(result, row)
			continue
		}

		if existing, ok := im.objects[entity][id]; ok {
			result = append(result, existing)
		} else {
			im.objects[entity][id] = row
			result = append(result, row)
		}
	}

	return result
}

// extractID returns the row identifier from the "id" field.
func (im *IdentityMap) extractID(row Row) string {
	id, ok := row["id"]
	if !ok {
		return ""
	}

	// Convert to a stable string key.
	switch v := id.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case [16]byte:
		return uuidToString(v)
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
