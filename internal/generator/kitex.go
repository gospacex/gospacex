package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// KitexGenerator Kitex 代码生成器
type KitexGenerator struct {
	OutputDir  string
	ModuleName string
}

// NewKitexGenerator creates a new KitexGenerator
func NewKitexGenerator(outputDir, moduleName string) *KitexGenerator {
	return &KitexGenerator{
		OutputDir:  outputDir,
		ModuleName: moduleName,
	}
}

// Generate generates Kitex code from Protobuf
func (g *KitexGenerator) Generate(protoFile string) error {
	// 检查 kitex 命令是否存在
	if _, err := exec.LookPath("kitex"); err != nil {
		return fmt.Errorf("kitex not found, please install: go install github.com/cloudwego/kitex/tool/cmd/kitex@latest")
	}

	apiDir := filepath.Join(g.OutputDir, "api")

	// 执行 kitex 命令
	cmd := exec.Command("kitex",
		"-module", g.ModuleName,
		"-I", apiDir,
		protoFile,
	)
	cmd.Dir = g.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kitex generate failed: %w", err)
	}

	return nil
}

// GenerateWithService generates Kitex code with custom service name
func (g *KitexGenerator) GenerateWithService(protoFile, serviceName string) error {
	if _, err := exec.LookPath("kitex"); err != nil {
		return fmt.Errorf("kitex not found, please install: go install github.com/cloudwego/kitex/tool/cmd/kitex@latest")
	}

	apiDir := filepath.Join(g.OutputDir, "api")

	cmd := exec.Command("kitex",
		"-module", g.ModuleName,
		"-service", serviceName,
		"-I", apiDir,
		protoFile,
	)
	cmd.Dir = g.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kitex generate failed: %w", err)
	}

	return nil
}
