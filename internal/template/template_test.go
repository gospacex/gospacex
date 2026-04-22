package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoader(t *testing.T) {
	t.Run("new loader", func(t *testing.T) {
		loader := NewLoader("./testdata")
		assert.NotNil(t, loader)
		assert.Equal(t, "./testdata", loader.baseDir)
	})
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_name", "userName"},
		{"User-Name", "userName"},
		{"USER_NAME", "uSERName"}, // 保留中间大写
		{"userName", "userName"},
		{"user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_name", "UserName"},
		{"User-Name", "UserName"},
		{"USER_NAME", "UserName"},
		{"userName", "Username"}, // 驼峰转 Pascal 会连在一起
		{"user", "User"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"userName", "user_name"},
		{"UserName", "user_name"},
		{"UserName", "user_name"},
		{"user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"userName", "user-name"},
		{"UserName", "user-name"},
		{"user", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToKebabCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefault(t *testing.T) {
	assert.Equal(t, "default", Default("", "default"))
	assert.Equal(t, "value", Default("value", "default"))
}

func TestTernary(t *testing.T) {
	assert.Equal(t, "true", Ternary(true, "true", "false"))
	assert.Equal(t, "false", Ternary(false, "true", "false"))
}

func TestDefaultMore(t *testing.T) {
	tests := []struct {
		name  string
		value string
		def   string
		want  string
	}{
		{"empty", "", "def", "def"},
		{"non-empty", "val", "def", "val"},
		{"both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Default(tt.value, tt.def))
		})
	}
}

func TestTernaryMore(t *testing.T) {
	tests := []struct {
		name   string
		cond   bool
		trueV  string
		falseV string
		want   string
	}{
		{"true", true, "a", "b", "a"},
		{"false", false, "a", "b", "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Ternary(tt.cond, tt.trueV, tt.falseV))
		})
	}
}

func TestDefaultFuncsContent(t *testing.T) {
	funcs := DefaultFuncs()
	assert.NotEmpty(t, funcs)

	_, ok := funcs["camelCase"]
	assert.True(t, ok)
	_, ok = funcs["default"]
	assert.True(t, ok)
}
