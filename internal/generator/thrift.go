package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ThriftGenerator Thrift IDL 生成器
type ThriftGenerator struct {
	OutputDir  string
	ModuleName string
}

// NewThriftGenerator 创建 Thrift 生成器
func NewThriftGenerator(outputDir, moduleName string) *ThriftGenerator {
	return &ThriftGenerator{
		OutputDir:  outputDir,
		ModuleName: moduleName,
	}
}

// Generate 生成 Thrift IDL 和代码
func (g *ThriftGenerator) Generate(serviceName string) error {
	if err := g.generateThriftIDL(serviceName); err != nil {
		return err
	}

	if err := g.generateKitexCode(serviceName); err != nil {
		return err
	}

	return nil
}

// generateThriftIDL 生成 Thrift IDL 文件
func (g *ThriftGenerator) generateThriftIDL(serviceName string) error {
	apiDir := filepath.Join(g.OutputDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return err
	}

	thriftContent := fmt.Sprintf(`namespace go %s

struct PingRequest {
    1: string message
}

struct PingResponse {
    1: string message
}

service %s {
    PingResponse ping(1: PingRequest req)
}
`, g.ModuleName, serviceName)

	thriftFile := filepath.Join(apiDir, serviceName+".thrift")
	return os.WriteFile(thriftFile, []byte(thriftContent), 0644)
}

// generateKitexCode 调用 kitex 生成代码
func (g *ThriftGenerator) generateKitexCode(serviceName string) error {
	if _, err := exec.LookPath("kitex"); err != nil {
		return fmt.Errorf("kitex not found, please install: go install github.com/cloudwego/kitex/tool/cmd/kitex@latest")
	}

	apiDir := filepath.Join(g.OutputDir, "api")

	cmd := exec.Command("kitex",
		"-module", g.ModuleName,
		"-I", apiDir,
		"-thrift",
		filepath.Join(apiDir, serviceName+".thrift"),
	)
	cmd.Dir = g.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kitex generate failed: %w", err)
	}

	return nil
}
