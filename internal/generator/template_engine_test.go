package generator

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"
)

// TestTemplateEngine 测试模板引擎基本功能
func TestTemplateEngine(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	
	t.Run("NewTemplateEngine", func(t *testing.T) {
		engine := NewTemplateEngine(tmpDir)
		if engine == nil {
			t.Error("Template engine should not be nil")
		}
		if engine.outputDir != tmpDir {
			t.Errorf("Expected outputDir %s, got %s", tmpDir, engine.outputDir)
		}
		if engine.templates == nil {
			t.Error("Templates map should be initialized")
		}
	})
}

// TestTemplateEngine_LoadTemplate 测试模板加载
func TestTemplateEngine_LoadTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewTemplateEngine(tmpDir)
	
	// 创建测试模板文件
	tmplContent := `package {{.PackageName}}

type {{.StructName}} struct {
	ID int
	Name string
}`
	
	tmplPath := filepath.Join(tmpDir, "test.go.tmpl")
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}
	
	t.Run("LoadTemplate_Success", func(t *testing.T) {
		err := engine.LoadTemplate("test", tmplPath)
		if err != nil {
			t.Errorf("LoadTemplate should succeed, got error: %v", err)
		}
		
		if _, ok := engine.templates["test"]; !ok {
			t.Error("Template should be loaded")
		}
	})
	
	t.Run("LoadTemplate_NotFound", func(t *testing.T) {
		err := engine.LoadTemplate("notfound", filepath.Join(tmpDir, "nonexistent.tmpl"))
		if err == nil {
			t.Error("LoadTemplate should fail for nonexistent file")
		}
	})
}

// TestTemplateEngine_LoadTemplates 测试批量加载模板
func TestTemplateEngine_LoadTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewTemplateEngine(tmpDir)
	
	// 创建测试模板目录结构
	dirs := []string{"pkg/config", "internal/model"}
	for _, dir := range dirs {
		fullDir := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}
		
		// 创建测试模板文件
		content := `// Template for ` + dir
		tmplPath := filepath.Join(fullDir, "test.go.tmpl")
		if err := os.WriteFile(tmplPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test template: %v", err)
		}
	}
	
	t.Run("LoadTemplates_FromDir", func(t *testing.T) {
		err := engine.LoadTemplates(tmpDir)
		if err != nil {
			t.Errorf("LoadTemplates should succeed, got error: %v", err)
		}
		
		// 验证模板数量
		if len(engine.templates) < 2 {
			t.Errorf("Expected at least 2 templates, got %d", len(engine.templates))
		}
	})
}

// TestTemplateEngine_Render 测试模板渲染
func TestTemplateEngine_Render(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewTemplateEngine(tmpDir)
	
	// 创建测试配置
	engine.config = &GeneratorConfig{
		PackageName:      "mypackage",
		StructName:       "MyStruct",
		EntityName:       "User",
		EntityNameLower:  "user",
	}
	
	// 创建测试模板
	tmplContent := `package {{.PackageName}}

type {{.StructName}} struct {
	ID int
	Name string
}`
	
	tmplPath := filepath.Join(tmpDir, "render_test.go.tmpl")
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}
	
	if err := engine.LoadTemplate("render_test", tmplPath); err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}
	
	outputPath := filepath.Join(tmpDir, "output", "test.go")
	
	t.Run("Render_Success", func(t *testing.T) {
		err := engine.Render("render_test", outputPath)
		if err != nil {
			t.Errorf("Render should succeed, got error: %v", err)
		}
		
		// 验证输出文件存在
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("Output file should exist")
		}
		
		// 验证输出内容
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}
		
		expectedContent := `package mypackage

type MyStruct struct {
	ID int
	Name string
}`
		
		if string(content) != expectedContent {
			t.Errorf("Rendered content mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, string(content))
		}
	})
	
	t.Run("Render_NotFound", func(t *testing.T) {
		err := engine.Render("nonexistent", outputPath)
		if err == nil {
			t.Error("Render should fail for nonexistent template")
		}
	})
}

// TestTemplateEngine_TemplateFunctions 测试模板函数
func TestTemplateEngine_TemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		expected string
	}{
		{"ToLowerCamelCase_User", ToLowerCamelCase, "UserProfile", "userProfile"},
		{"ToLowerCamelCase_ID", ToLowerCamelCase, "UserID", "userID"},
		{"ToSnakeCase_User", ToSnakeCase, "UserProfile", "user_profile"},
		{"ToSnakeCase_HTTP", ToSnakeCase, "HTTPServer", "h_t_t_p_server"},
		{"ToPascalCase_user", ToPascalCase, "userProfile", "UserProfile"},
		{"ToPascalCase_http_server", ToPascalCase, "http_server", "HttpServer"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			if result != tt.expected {
				t.Errorf("%s(%q) = %q, want %q", getFuncName(tt.fn), tt.input, result, tt.expected)
			}
		})
	}
}

// TestTemplateEngine_WithCustomFuncs 测试自定义模板函数
func TestTemplateEngine_WithCustomFuncs(t *testing.T) {
	tmpDir := t.TempDir()
	engine := NewTemplateEngine(tmpDir)
	
	// 创建使用自定义函数的模板
	tmplContent := `package {{.PackageName}}

// {{.StructName}} represents {{ToLowerCamelCase .StructName}}
type {{.StructName}} struct {
	// ID is the identifier
	ID int
}`
	
	tmplPath := filepath.Join(tmpDir, "func_test.go.tmpl")
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}
	
	// 创建带自定义函数的模板引擎
	funcMap := template.FuncMap{
		"ToLowerCamelCase": ToLowerCamelCase,
	}
	
	tmpl, err := template.New("func_test.go.tmpl").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	
	engine.templates["func_test"] = tmpl
	engine.config = &GeneratorConfig{
		PackageName: "testpkg",
		StructName:  "MyEntity",
	}
	
	outputPath := filepath.Join(tmpDir, "output", "func_test.go")
	err = engine.Render("func_test", outputPath)
	if err != nil {
		t.Errorf("Render with custom funcs should succeed, got error: %v", err)
	}
	
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	
	// 验证自定义函数被正确调用
	expectedSubstring := "// MyEntity represents myEntity"
	if !contains(string(content), expectedSubstring) {
		t.Errorf("Expected custom function to be applied. Got:\n%s", string(content))
	}
}

// Helper functions
func getFuncName(fn interface{}) string {
	switch fn.(type) {
	case func(string) string:
		// 通过测试用例名称推断
		return "function"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
