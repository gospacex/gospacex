package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// ProtobufGenerator generates Protobuf IDL files
type ProtobufGenerator struct {
	OutputDir string
}

// NewProtobufGenerator creates a new ProtobufGenerator
func NewProtobufGenerator(outputDir string) *ProtobufGenerator {
	return &ProtobufGenerator{
		OutputDir: outputDir,
	}
}

// Generate creates the Protobuf file from template
func (g *ProtobufGenerator) Generate(serviceName, packageName, moduleName string) error {
	tmplContent := `syntax = "proto3";

package {{ .PackageName }};

option go_package = "{{ .ModuleName }}/api/kitex_gen/{{ .ServiceName }}";

service {{ .ServiceName | camelcase }} {
    rpc Ping (PingRequest) returns (PingResponse);
}

message PingRequest {
    string message = 1;
}

message PingResponse {
    string message = 1;
}
`

	tmpl, err := template.New("proto").Funcs(template.FuncMap{
		"camelcase": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return string(s[0]-32) + s[1:]
		},
	}).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	apiDir := filepath.Join(g.OutputDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("create api dir: %w", err)
	}

	file, err := os.Create(filepath.Join(apiDir, serviceName+".proto"))
	if err != nil {
		return fmt.Errorf("create proto file: %w", err)
	}
	defer file.Close()

	data := map[string]string{
		"ServiceName": serviceName,
		"PackageName": packageName,
		"ModuleName":  moduleName,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
