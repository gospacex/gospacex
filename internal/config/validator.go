package config

import (
	"fmt"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate 验证配置
func (v *ConfigValidator) Validate(cfg *ProjectConfig) error {
	// 规则 1: Istio 架构不支持服务注册
	if cfg.Style == "istio" && cfg.Registry != "" {
		return fmt.Errorf("Istio 架构不支持服务注册 (--registry)")
	}

	// 规则 2: Thrift IDL 仅支持微服务项目
	if cfg.IDL == "thrift" && cfg.ProjectType != "microservice" {
		return fmt.Errorf("Thrift IDL 仅支持微服务项目")
	}

	// 规则 3: 单体项目不支持 IDL
	if cfg.ProjectType == "monolith" && cfg.IDL != "" {
		return fmt.Errorf("单体项目不支持 IDL")
	}

	// 规则 4: 脚本中心不支持 IDL
	if cfg.ProjectType == "script" && cfg.IDL != "" {
		return fmt.Errorf("脚本中心不支持 IDL")
	}

	// 规则 5: Agent 项目不支持 IDL
	if cfg.ProjectType == "agent" && cfg.IDL != "" {
		return fmt.Errorf("Agent 项目不支持 IDL")
	}

	return nil
}
