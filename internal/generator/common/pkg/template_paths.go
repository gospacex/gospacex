package pkg

import (
	"os"
	"path/filepath"
	"runtime"
)

func packageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func projectRootDir() string {
	dir := packageDir()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Clean(filepath.Join(packageDir(), "..", "..", "..", ".."))
		}
		dir = parent
	}
}

func resolveTemplatePath(relPath string) string {
	return filepath.Join(projectRootDir(), "templates", relPath)
}
