package hcl

import (
	"io/fs"
	"os"
	"strings"
)

// Walk recursively walks a directory, finds all *.tf files, parses each one,
// and returns a list of TFModules. Skips .terraform/ directories.
//
// scan-fix(gosec:G122): reads go through an os.Root (Go 1.24+) scoped to dir
// instead of a plain filepath.WalkDir + os.ReadFile(absolutePath) pair. dir is
// a freshly `git clone`d, potentially untrusted repo (see
// scanner.TerraformScanner.Scan) — a maliciously named file (e.g. "evil.tf"
// symlinked to /etc/passwd or a secrets file) could otherwise be read via a
// symlink. Root resolves every path relative to dir using the OS's
// openat-family syscalls and rejects any symlink that would escape dir,
// atomically at open time — no separate stat-then-read TOCTOU window like a
// manual d.Type()-then-os.ReadFile check would have.
func Walk(dir string) ([]TFModule, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	var modules []TFModule

	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .terraform directories
		if d.IsDir() && d.Name() == ".terraform" {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(p, ".tf") {
			return nil
		}

		content, err := root.ReadFile(p)
		if err != nil {
			return err
		}

		// p is already relative to dir (fs.WalkDir paths are root-relative).
		module, _ := ParseFile(p, string(content))
		if module != nil {
			modules = append(modules, *module)
		}

		return nil
	})

	return modules, err
}
