package mask

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
)

type Masker struct {
	rules map[string]*regexp.Regexp
}

type MaskConfig struct {
	PhonePattern    string
	IDCardPattern   string
	BankCardPattern string
	EmailPattern    string
}

func DefaultMaskConfig() MaskConfig {
	return MaskConfig{
		PhonePattern:    `(\d{3})\d{4}(\d{4})`,
		IDCardPattern:   `(\d{6})\d{8}(\d{4})`,
		BankCardPattern: `(\d{4})\d+(\d{4})`,
		EmailPattern:    `(\w+)@(\w+)\.(\w+)`,
	}
}

func NewMasker(cfg MaskConfig) *Masker {
	if cfg.PhonePattern == "" {
		cfg = DefaultMaskConfig()
	}

	return &Masker{
		rules: map[string]*regexp.Regexp{
			"phone":    regexp.MustCompile(cfg.PhonePattern),
			"idcard":   regexp.MustCompile(cfg.IDCardPattern),
			"bankcard": regexp.MustCompile(cfg.BankCardPattern),
			"email":    regexp.MustCompile(cfg.EmailPattern),
		},
	}
}

func (m *Masker) Mask(data map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range data {
		result[key] = m.maskRecursive(value, 0)
	}
	return result
}

func (m *Masker) maskRecursive(data any, depth int) any {
	if depth > 5 {
		return "[max depth reached]"
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			result[key] = m.maskRecursive(value, depth+1)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = m.maskRecursive(item, depth+1)
		}
		return result
	case string:
		return v
	default:
		return v
	}
}

func (m *Masker) MaskField(value string, maskType string) string {
	re, ok := m.rules[maskType]
	if !ok {
		return value
	}

	switch maskType {
	case "phone":
		return re.ReplaceAllString(value, "$1****$2")
	case "idcard":
		return re.ReplaceAllString(value, "$1********$2")
	case "bankcard":
		return re.ReplaceAllString(value, "$1****$2")
	case "email":
		idx := strings.Index(value, "@")
		if idx <= 1 {
			return value
		}
		username := value[:idx]
		domain := value[idx:]
		if len(username) < 3 {
			return string(username[0]) + "***" + domain
		}
		return username[:2] + "***" + domain
	}
	return value
}

type maskable interface {
	Mask(masker *Masker) any
}

func MaskStruct[T any](data T, masker *Masker) T {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Struct {
		return data
	}

	result := reflect.New(val.Type()).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		sf := val.Type().Field(i)

		maskTag := sf.Tag.Get("mask")
		if maskTag != "" && field.Kind() == reflect.String {
			masked := masker.MaskField(field.String(), maskTag)
			result.Field(i).SetString(masked)
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			masked := MaskStruct(field.Elem().Interface(), masker)
			result.Field(i).Set(reflect.ValueOf(&masked))
		} else {
			result.Field(i).Set(field)
		}
	}
	return result.Interface().(T)
}

func MaskJSONString(data string, masker *Masker) (string, error) {
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
		return data, err
	}
	masked := masker.Mask(jsonData)
	result, err := json.Marshal(masked)
	if err != nil {
		return data, err
	}
	return string(result), nil
}
