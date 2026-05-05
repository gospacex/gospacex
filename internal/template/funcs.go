package template

import (
	"strings"
	"text/template"
	"unicode"
)

// DefaultFuncs 返回默认模板函数
func DefaultFuncs() template.FuncMap {
	return template.FuncMap{
		// 字符串转换
		"toLower":    strings.ToLower,
		"toUpper":    strings.ToUpper,
		"toTitle":    strings.ToTitle,
		"trimSpace":  strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"join":       strings.Join,

		// 命名转换
		"camelCase":  ToCamelCase,
		"pascalCase": ToPascalCase,
		"snakeCase":  ToSnakeCase,
		"kebabCase":  ToKebabCase,

		// 其他
		"default": Default,
		"ternary": Ternary,
	}
}

// ToCamelCase 转换为驼峰命名 (camelCase)
func ToCamelCase(s string) string {
	return toCamelCase(s, false)
}

// ToPascalCase 转换为帕斯卡命名 (PascalCase)
func ToPascalCase(s string) string {
	return toCamelCase(s, true)
}

func toCamelCase(s string, upperFirst bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	if len(words) == 0 {
		return s
	}

	var result strings.Builder
	for i, word := range words {
		if i == 0 && !upperFirst {
			result.WriteString(strings.ToLower(string(word[0])))
			if len(word) > 1 {
				result.WriteString(word[1:])
			}
		} else {
			result.WriteString(strings.ToUpper(string(word[0])))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}

	return result.String()
}

// ToSnakeCase 转换为蛇形命名 (snake_case)
func ToSnakeCase(s string) string {
	return toSeparatedCase(s, '_')
}

// ToKebabCase 转换为短横线命名 (kebab-case)
func ToKebabCase(s string) string {
	return toSeparatedCase(s, '-')
}

func toSeparatedCase(s string, sep rune) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune(sep)
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Default 返回默认值（如果 value 为空）
func Default(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// Ternary 三元运算符
func Ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
