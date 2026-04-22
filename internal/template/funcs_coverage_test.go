package template

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestStringHelpers(t *testing.T) {
	assert.Equal(t, "hello", strings.ToLower("HELLO"))
	assert.Equal(t, "HELLO", strings.ToUpper("hello"))
	assert.Equal(t, "hello", strings.TrimSpace("  hello  "))
	assert.True(t, strings.Contains("hello", "ell"))
	assert.True(t, strings.HasPrefix("hello", "he"))
	assert.True(t, strings.HasSuffix("hello", "lo"))
	assert.Equal(t, "heLLo", strings.ReplaceAll("hello", "l", "L"))
	assert.Equal(t, []string{"a", "b"}, strings.Split("a,b", ","))
	assert.Equal(t, "a-b", strings.Join([]string{"a", "b"}, "-"))
}

func TestTemplateFuncsIntegration(t *testing.T) {
	funcs := DefaultFuncs()
	assert.NotNil(t, funcs)

	tmpl, err := template.New("test").Funcs(funcs).Parse(`{{toLower "HELLO"}}`)
	assert.NoError(t, err)

	var buf strings.Builder
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "hello", buf.String())
}

func TestToSnakeCaseComprehensive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user_name"},
		{"", ""},
		{"simple", "simple"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ToSnakeCase(tt.input))
		})
	}
}

func TestToKebabCaseComprehensive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"userName", "user-name"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ToKebabCase(tt.input))
		})
	}
}

func TestToCamelCaseEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "a"},
		{"AB", "aB"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ToCamelCase(tt.input))
		})
	}
}

func TestToPascalCaseEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"a", "A"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ToPascalCase(tt.input))
		})
	}
}

func TestDefaultEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		def    string
		expect string
	}{
		{"both empty", "", "", ""},
		{"value space", " ", "def", " "},
		{"def space", "", " ", " "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, Default(tt.value, tt.def))
		})
	}
}

func TestTernaryEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		cond   bool
		trueV  string
		falseV string
		expect string
	}{
		{"both empty true", true, "", "", ""},
		{"both empty false", false, "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, Ternary(tt.cond, tt.trueV, tt.falseV))
		})
	}
}
