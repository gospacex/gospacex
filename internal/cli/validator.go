package cli

import (
	"fmt"
)

var validProjectTypes = []string{
	"microservice",
	"monolith",
	"script",
	"agent",
}

// validateProjectType 验证项目类型是否有效
func validateProjectType(projectType string) error {
	for _, valid := range validProjectTypes {
		if projectType == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid project type: %s (valid: %v)", projectType, validProjectTypes)
}

// validateStyle 验证架构风格
func validateStyle(style string) error {
	validStyles := []string{"standard", "ddd", "istio"}
	for _, valid := range validStyles {
		if style == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid style: %s (valid: %v)", style, validStyles)
}

// validateIDL 验证 IDL 类型
func validateIDL(idl string) error {
	validIDLs := []string{"protobuf", "thrift"}
	for _, valid := range validIDLs {
		if idl == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid idl: %s (valid: %v)", idl, validIDLs)
}

// validateORM 验证 ORM 框架
func validateORM(orm string) error {
	validORMs := []string{"gorm", "xorm"}
	for _, valid := range validORMs {
		if orm == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid orm: %s (valid: %v)", orm, validORMs)
}
