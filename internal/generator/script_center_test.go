package generator

import (
	"os"
	"testing"

	"github.com/gospacex/gpx/internal/config"
)

func TestScriptCenterGenerator(t *testing.T) {
	cfg := config.NewProjectConfig()
	cfg.ProjectType = "script"
	cfg.OutputDir = "/tmp/test-scripts"

	g := NewScriptCenterGenerator(cfg)
	err := g.Generate("/tmp/test-scripts")
	if err != nil {
		t.Fatal(err)
	}

	// Check generated files (matching actual generator implementation)
	files := []string{
		"cmd/main.go",
		"cmd/commands/root.go",
		"cmd/commands/start.go",
		"pkg/config/config.go",
		"pkg/config/types.go",
		"pkg/logger/logger.go",
		"configs/config.yaml",
		"deploy/supervisor/app.conf",
		"deploy/systemd/service.service",
		"scripts/run.sh",
		"Makefile",
		"readme.md",
		".gitignore",
	}

	for _, file := range files {
		path := "/tmp/test-scripts/" + file
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File not generated: %s", path)
		}
	}
}
