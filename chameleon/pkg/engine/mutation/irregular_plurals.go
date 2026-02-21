package mutation

import "strings"

// Irregular plurals
var irregularPlurals = map[string]string{
	"person":     "people",
	"child":      "children",
	"tooth":      "teeth",
	"foot":       "feet",
	"mouse":      "mice",
	"goose":      "geese",
	"man":        "men",
	"woman":      "women",
	"datum":      "data",
	"medium":     "media",
	"index":      "indices",
	"matrix":     "matrices",
	"vertex":     "vertices",
	"axis":       "axes",
	"analysis":   "analyses",
	"basis":      "bases",
	"crisis":     "crises",
	"thesis":     "theses",
	"diagnosis":  "diagnoses",
	"synopsis":   "synopses",
	"criterion":  "criteria",
	"phenomenon": "phenomena",
	"radius":     "radii",
	"formula":    "formulae",
	"focus":      "foci",
	"nucleus":    "nuclei",
	"syllabus":   "syllabi",
	"curriculum": "curricula",
	"leaf":       "leaves",
	"life":       "lives",
	"knife":      "knives",
	"wife":       "wives",
	"self":       "selves",
	"half":       "halves",
	"loaf":       "loaves",
	"calf":       "calves",
	"hero":       "heroes",
	"potato":     "potatoes",
	"tomato":     "tomatoes",
	"echo":       "echoes",
	"sheep":      "sheep",
	"fish":       "fish",
	"series":     "series",
	"species":    "species",
	"status":     "statuses",
	"alias":      "aliases",
	"bus":        "buses",
}

var irregularSingulars = func() map[string]string {
	result := make(map[string]string, len(irregularPlurals))
	for singular, plural := range irregularPlurals {
		result[plural] = singular
	}
	return result
}()

// SingularizeName converts plural names to singular.
// It uses irregular plural mappings first; if not found and the name ends in 's', it removes the trailing 's'.
func SingularizeName(name string) string {
	if name == "" {
		return name
	}

	lower := strings.ToLower(name)
	if singular, ok := irregularSingulars[lower]; ok {
		return applyWordCase(name, singular)
	}

	if len(name) > 1 && strings.HasSuffix(lower, "s") {
		return name[:len(name)-1]
	}

	return name
}

func applyWordCase(original string, replacement string) string {
	if original == strings.ToUpper(original) {
		return strings.ToUpper(replacement)
	}

	if len(original) > 0 && original[:1] == strings.ToUpper(original[:1]) {
		return strings.ToUpper(replacement[:1]) + replacement[1:]
	}

	return replacement
}
