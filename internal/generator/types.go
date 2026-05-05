package generator

// FieldConfig 字段配置
type FieldConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Column   string `yaml:"column"`
	DbType   string `yaml:"db_type"`
	JsonName string `yaml:"json_name"`
	Default  string `yaml:"default"`
	NotNull  bool   `yaml:"not_null"`
}

// DataSourceConfig 数据源配置
type DataSourceConfig struct {
	MySQL         bool `yaml:"mysql"`
	Redis         bool `yaml:"redis"`
	Elasticsearch bool `yaml:"elasticsearch"`
	MongoDB       bool `yaml:"mongodb"`
}

// GeneratorConfig 生成器配置
type GeneratorConfig struct {
	ProjectName     string            `yaml:"project_name"`
	PackageName     string            `yaml:"package_name"`
	StructName      string            `yaml:"struct_name"`
	EntityName      string            `yaml:"entity_name"`
	EntityNameLower string            `yaml:"entity_name_lower"`
	TableName       string            `yaml:"table_name"`
	Fields          []FieldConfig     `yaml:"fields"`
	DataSources     DataSourceConfig  `yaml:"datasources"`
}

// ToLowerCamelCase converts string to lowerCamelCase
func ToLowerCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]|0x20) + s[1:]
}
