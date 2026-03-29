package hcl

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Walk recursively walks a directory, finds all *.tf files, parses each one,
// and returns a list of TFModules. Skips .terraform/ directories.
func Walk(dir string) ([]TFModule, error) {
	var modules []TFModule

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .terraform directories
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(p, ".tf") {
			return nil
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}

		// Get relative path from dir
		relPath, err := filepath.Rel(dir, p)
		if err != nil {
			relPath = p
		}

		module, _ := ParseFile(relPath, string(content))
		if module != nil {
			modules = append(modules, *module)
		}

		return nil
	})

	return modules, err
}
