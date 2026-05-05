package variables

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator 变量验证器
type Validator struct {
	projectNamePattern *regexp.Regexp
	fileNamePattern    *regexp.Regexp
}

// ValidationError 验证错误
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %q: %s", e.Field, e.Msg)
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	return &Validator{
		// Go 模块名允许: 字母、数字、斜杠、连字符、下划线、点
		projectNamePattern: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9/_\-/.]*[a-zA-Z0-9]$`),
		// 文件名允许: 字母、数字、连字符、下划线、点，不能以点开头
		fileNamePattern: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-/.]*$`),
	}
}

// ValidateProjectName 验证项目名称（Go 模块名）
func (v *Validator) ValidateProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ValidationError{Field: "project_name", Msg: "cannot be empty"}
	}

	if !v.projectNamePattern.MatchString(name) {
		return &ValidationError{
			Field: "project_name",
			Msg:   fmt.Sprintf("invalid characters: %q (must be valid Go module name)", name),
		}
	}

	return nil
}

// ValidateOutputDir 验证输出目录
func (v *Validator) ValidateOutputDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return &ValidationError{Field: "output_dir", Msg: "cannot be empty"}
	}

	// 检查路径是否包含危险字符
	if strings.Contains(dir, "..") {
		return &ValidationError{Field: "output_dir", Msg: "cannot contain '..' for security"}
	}

	return nil
}

// ValidateDatabaseTypes 验证数据库类型列表
func (v *Validator) ValidateDatabaseTypes(dbs []string) error {
	validTypes := map[string]bool{
		"mysql":         true,
		"postgres":      true,
		"postgresql":    true,
		"sqlite":        true,
		"sqlite3":       true,
		"redis":         true,
		"mongodb":       true,
		"mongo":         true,
		"elasticsearch": true,
		"es":            true,
		"etcd":          true,
	}

	for _, db := range dbs {
		db = strings.ToLower(strings.TrimSpace(db))
		if db == "" {
			continue
		}
		if !validTypes[db] {
			return &ValidationError{
				Field: "database_types",
				Msg:   fmt.Sprintf("unsupported database type: %q", db),
			}
		}
	}

	return nil
}

// ValidateMessageQueue 验证消息队列类型
func (v *Validator) ValidateMessageQueue(mq string) error {
	if mq == "" {
		return nil // 可选
	}

	validTypes := map[string]bool{
		"kafka":    true,
		"rabbitmq": true,
		"rabbit":   true,
		"rocketmq": true,
	}

	mq = strings.ToLower(strings.TrimSpace(mq))
	for _, t := range strings.Split(mq, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !validTypes[t] {
			return &ValidationError{
				Field: "message_queue",
				Msg:   fmt.Sprintf("unsupported message queue type: %q", t),
			}
		}
	}

	return nil
}

// ValidateRegistry 验证服务注册中心类型
func (v *Validator) ValidateRegistry(registry string) error {
	if registry == "" {
		return nil // 可选
	}

	validTypes := map[string]bool{
		"etcd":   true,
		"nacos":  true,
		"consul": true,
	}

	registry = strings.ToLower(strings.TrimSpace(registry))
	if !validTypes[registry] {
		return &ValidationError{
			Field: "registry",
			Msg:   fmt.Sprintf("unsupported registry type: %q", registry),
		}
	}

	return nil
}

// ValidateConfigCenter 验证配置中心类型
func (v *Validator) ValidateConfigCenter(config string) error {
	if config == "" {
		return nil // 可选
	}

	validTypes := map[string]bool{
		"nacos":  true,
		"apollo": true,
		"consul": true,
	}

	config = strings.ToLower(strings.TrimSpace(config))
	if !validTypes[config] {
		return &ValidationError{
			Field: "config_center",
			Msg:   fmt.Sprintf("unsupported config center type: %q", config),
		}
	}

	return nil
}

// ValidateMicroserviceStyle 验证微服务架构风格
func (v *Validator) ValidateMicroserviceStyle(style string) error {
	style = strings.ToLower(strings.TrimSpace(style))
	validStyles := map[string]bool{
		"simple":       true,
		"standard":     true,
		"full":         true,
		"ddd":          true,
		"clean":        true,
		"clean-arch":   true,
		"istio":        true,
		"service-mesh": true,
	}

	if !validStyles[style] {
		return &ValidationError{
			Field: "microservice_style",
			Msg:   fmt.Sprintf("unsupported style: %q (valid: simple, full, ddd, clean-arch, istio)", style),
		}
	}

	return nil
}

// Validate 验证所有配置
func (v *Validator) Validate(projectName, outputDir string, dbs []string, mq string, registry string, config string, style string) error {
	if err := v.ValidateProjectName(projectName); err != nil {
		return err
	}
	if err := v.ValidateOutputDir(outputDir); err != nil {
		return err
	}
	if err := v.ValidateDatabaseTypes(dbs); err != nil {
		return err
	}
	if err := v.ValidateMessageQueue(mq); err != nil {
		return err
	}
	if err := v.ValidateRegistry(registry); err != nil {
		return err
	}
	if err := v.ValidateConfigCenter(config); err != nil {
		return err
	}
	if style != "" {
		if err := v.ValidateMicroserviceStyle(style); err != nil {
			return err
		}
	}

	return nil
}
