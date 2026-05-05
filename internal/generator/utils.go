package generator

import "strings"

const GoVersion = "1.26.2"

func GetGoVersion() string {
	return GoVersion
}

// ParseFields parses field string (format: Name:Type,Name:Type)
func ParseFields(s string) []FieldConfig {
	if s == "" {
		return []FieldConfig{}
	}

	fields := []FieldConfig{}
	for _, fieldStr := range strings.Split(s, ",") {
		parts := strings.Split(fieldStr, ":")
		if len(parts) >= 2 {
			field := FieldConfig{
				Name: parts[0],
				Type: parts[1],
			}

			if len(parts) >= 3 {
				field.Column = parts[2]
			} else {
				field.Column = toSnakeCase(parts[0])
			}

			if len(parts) >= 4 {
				field.DbType = parts[3]
			}

			field.JsonName = ToLowerCamelCase(parts[0])

			if len(parts) >= 5 {
				field.Default = parts[4]
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// ParseDatasources parses datasource string (format: mysql,redis,es,mongo)
func ParseDatasources(s string) DataSourceConfig {
	ds := DataSourceConfig{}

	for _, dsName := range strings.Split(s, ",") {
		dsName = strings.TrimSpace(strings.ToLower(dsName))
		switch dsName {
		case "mysql":
			ds.MySQL = true
		case "redis":
			ds.Redis = true
		case "elasticsearch", "es":
			ds.Elasticsearch = true
		case "mongodb", "mongo":
			ds.MongoDB = true
		}
	}

	return ds
}

// ToSnakeCase converts PascalCase or camelCase to snake_case (exported)
func ToSnakeCase(s string) string {
	if len(s) == 0 {
		return s
	}

	result := ""
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		result += string(r | 0x20)
	}
	return result
}

// toSnakeCase converts string to snake_case (alias for ToSnakeCase)
func toSnakeCase(s string) string {
	return ToSnakeCase(s)
}

// ToPascalCase converts snake_case or lowerCamelCase to PascalCase
func ToPascalCase(s string) string {
	// Handle empty string
	if s == "" {
		return s
	}

	// If contains underscore, split by underscore
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += strings.ToUpper(string(part[0])) + part[1:]
			}
		}
		return result
	}

	// Otherwise, just capitalize first letter
	return strings.ToUpper(string(s[0])) + s[1:]
}

// ToGoFieldName converts snake_case to PascalCase (alias for ToPascalCase)
func ToGoFieldName(s string) string {
	return ToPascalCase(s)
}

// MapDbTypeToGoType maps database type to Go type
func MapDbTypeToGoType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "bigint", "int", "mediumint", "smallint", "tinyint":
		return "int64"
	case "varchar", "text", "char", "tinytext", "mediumtext", "longtext":
		return "string"
	case "datetime", "timestamp", "date":
		return "time.Time"
	case "decimal", "float", "double":
		return "float64"
	case "bit", "boolean":
		return "bool"
	default:
		return "string"
	}
}
