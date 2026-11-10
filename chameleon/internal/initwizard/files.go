package initwizard

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileIfAbsent writes content to path unless something is already there.
// It returns created=false, with no error, when the path exists. `init` uses it
// for every file it writes, so that running it in a directory with work in it
// never replaces that work.
func WriteFileIfAbsent(path string, content []byte, perm os.FileMode) (created bool, err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
	}

	if _, err := f.Write(content); err != nil {
		f.Close()
		return false, fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
	}

	if err := f.Close(); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
	}

	return true, nil
}

// HasSchemaFiles reports whether dir contains a .cham file at any depth.
func HasSchemaFiles(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".cham") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
