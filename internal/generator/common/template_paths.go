package common

import (
	"os"
	"path/filepath"
	"runtime"
)

func commonPackageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

func projectRootDir() string {
	dir := commonPackageDir()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Clean(filepath.Join(commonPackageDir(), "..", "..", ".."))
		}
		dir = parent
	}
}

func projectTemplateDir() string {
	return filepath.Join(projectRootDir(), "templates")
}

func resolveTemplatePath(parts ...string) string {
	relPath := filepath.Join(parts...)
	candidates := []string{
		filepath.Join(commonPackageDir(), "templates", relPath),
		filepath.Join(projectTemplateDir(), relPath),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return candidates[len(candidates)-1]
}
