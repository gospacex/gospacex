package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestOtelFlagInHelp(t *testing.T) {
	cmd := newMicroAppCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--otel") {
		t.Errorf("Expected --otel flag to be present in help output, got:\n%s", output)
	}
}

func TestTemplateDataOtelField(t *testing.T) {
	tests := []struct {
		name     string
		otel     bool
		wantOtel bool
	}{
		{
			name:     "otel disabled by default",
			otel:     false,
			wantOtel: false,
		},
		{
			name:     "otel enabled with flag",
			otel:     true,
			wantOtel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"Otel": tt.otel,
			}

			tmplStr := `{{if .Otel}}otel enabled{{else}}otel disabled{{end}}`
			tmpl, err := template.New("").Parse(tmplStr)
			if err != nil {
				t.Fatalf("parse template failed: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			result := buf.String()
			if tt.wantOtel {
				if !strings.Contains(result, "otel enabled") {
					t.Errorf("Expected 'otel enabled' in output, got: %s", result)
				}
			} else {
				if !strings.Contains(result, "otel disabled") {
					t.Errorf("Expected 'otel disabled' in output, got: %s", result)
				}
			}
		})
	}
}

func TestGinMainTemplateOtelCondition(t *testing.T) {
	otelImport := `"go.opentelemetry.io/otel"`
	otelInit := `otel.SetTracerProvider`

	tests := []struct {
		name       string
		otel       bool
		wantImport bool
		wantInit   bool
	}{
		{
			name:       "without otel flag",
			otel:       false,
			wantImport: false,
			wantInit:   false,
		},
		{
			name:       "with otel flag",
			otel:       true,
			wantImport: true,
			wantInit:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"Otel": tt.otel,
			}

			tmplStr := `package main
{{if .Otel}}
import (
	"go.opentelemetry.io/otel"
)

func init() {
	otel.SetTracerProvider(nil)
}
{{end}}`

			tmpl, err := template.New("").Parse(tmplStr)
			if err != nil {
				t.Fatalf("parse template failed: %v", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			result := buf.String()

			if tt.wantImport {
				if !strings.Contains(result, otelImport) {
					t.Errorf("Expected OTel import in output when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelImport) {
					t.Errorf("Expected NO OTel import in output when Otel=false, got:\n%s", result)
				}
			}

			if tt.wantInit {
				if !strings.Contains(result, otelInit) {
					t.Errorf("Expected OTel init in output when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelInit) {
					t.Errorf("Expected NO OTel init in output when Otel=false, got:\n%s", result)
				}
			}
		})
	}
}

func TestGinMiddlewareTemplateOtelCondition(t *testing.T) {
	otelImport := `"go.opentelemetry.io/otel"`
	otelTrace := `trace.SpanContextFromContext`

	tests := []struct {
		name        string
		otel        bool
		wantImport  bool
		wantTrace   bool
	}{
		{
			name:        "without otel flag",
			otel:        false,
			wantImport:  false,
			wantTrace:   false,
		},
		{
			name:        "with otel flag",
			otel:        true,
			wantImport:  true,
			wantTrace:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "bff", "middleware", "gin_middleware.go.tmpl")
			tmplBytes, err := os.ReadFile(tmplPath)
			if err != nil {
				t.Fatalf("read template file failed: %v", err)
			}

			data := map[string]interface{}{
				"AppName": "testapp",
				"BFFName": "h5",
				"Otel":    tt.otel,
			}

			result, err := executeTemplate(string(tmplBytes), data)
			if err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			if tt.wantImport {
				if !strings.Contains(result, otelImport) {
					t.Errorf("Expected OTel import in middleware when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelImport) {
					t.Errorf("Expected NO OTel import in middleware when Otel=false, got:\n%s", result)
				}
			}

			if tt.wantTrace {
				if !strings.Contains(result, otelTrace) {
					t.Errorf("Expected trace.SpanContextFromContext in middleware when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelTrace) {
					t.Errorf("Expected NO trace.SpanContextFromContext in middleware when Otel=false, got:\n%s", result)
				}
			}
		})
	}
}

func TestGrpcInterceptorTemplateOtelCondition(t *testing.T) {
	otelImport := `"go.opentelemetry.io/otel"`
	otelTracer := `otel.Tracer`

	tests := []struct {
		name        string
		otel        bool
		wantImport  bool
		wantTracer  bool
	}{
		{
			name:        "without otel flag",
			otel:        false,
			wantImport:  false,
			wantTracer:  false,
		},
		{
			name:        "with otel flag",
			otel:        true,
			wantImport:  true,
			wantTracer:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "srv", "interceptor", "grpc_interceptor.go.tmpl")
			tmplBytes, err := os.ReadFile(tmplPath)
			if err != nil {
				t.Fatalf("read template file failed: %v", err)
			}

			data := map[string]interface{}{
				"AppName": "testapp",
				"Module":  "product",
				"Otel":    tt.otel,
			}

			result, err := executeTemplate(string(tmplBytes), data)
			if err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			if tt.wantImport {
				if !strings.Contains(result, otelImport) {
					t.Errorf("Expected OTel import in interceptor when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelImport) {
					t.Errorf("Expected NO OTel import in interceptor when Otel=false, got:\n%s", result)
				}
			}

			if tt.wantTracer {
				if !strings.Contains(result, otelTracer) {
					t.Errorf("Expected otel.Tracer in interceptor when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelTracer) {
					t.Errorf("Expected NO otel.Tracer in interceptor when Otel=false, got:\n%s", result)
				}
			}
		})
	}
}

func TestSrvMainTemplateOtelCondition(t *testing.T) {
	otelImport := `"go.opentelemetry.io/otel"`
	otelInit := `otel.SetTracerProvider`

	tests := []struct {
		name        string
		otel        bool
		wantImport  bool
		wantInit    bool
	}{
		{
			name:        "without otel flag",
			otel:        false,
			wantImport:  false,
			wantInit:    false,
		},
		{
			name:        "with otel flag",
			otel:        true,
			wantImport:  true,
			wantInit:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmplPath := filepath.Join(getTemplatesDir(), "micro-app", "srv", "main", "main_direct.go.tmpl")
			tmplBytes, err := os.ReadFile(tmplPath)
			if err != nil {
				t.Fatalf("read template file failed: %v", err)
			}

			data := map[string]interface{}{
				"AppName":     "testapp",
				"Module":      "product",
				"UpperModule": "Product",
				"SrvDirName":  "srvProduct",
				"Otel":        tt.otel,
			}

			result, err := executeTemplate(string(tmplBytes), data)
			if err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			if tt.wantImport {
				if !strings.Contains(result, otelImport) {
					t.Errorf("Expected OTel import in srv main when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelImport) {
					t.Errorf("Expected NO OTel import in srv main when Otel=false, got:\n%s", result)
				}
			}

			if tt.wantInit {
				if !strings.Contains(result, otelInit) {
					t.Errorf("Expected otel.SetTracerProvider in srv main when Otel=true, got:\n%s", result)
				}
			} else {
				if strings.Contains(result, otelInit) {
					t.Errorf("Expected NO otel.SetTracerProvider in srv main when Otel=false, got:\n%s", result)
				}
			}
		})
	}
}
