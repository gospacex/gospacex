package generator

import (
	"fmt"
	"strings"
	"time"
)

// TableInfo 数据库表结构信息
type TableInfo struct {
	TableName    string       // 表名
	Comment      string       // 表注释
	Columns      []ColumnInfo // 列信息
	PrimaryKey   string       // 主键列名
	CreatedAt    string       // 创建时间字段
	UpdatedAt    string       // 更新时间字段
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string // 列名
	Type         string // 数据类型 (Go 类型)
	ProtoType    string // Proto 类型
	OriginalType string // 原始数据库类型
	Comment      string // 列注释
	IsPrimary    bool   // 是否主键
	IsNullable   bool   // 是否可为空
	DefaultValue string // 默认值
}

// ProtoGenerator Proto文件生成器
type ProtoGenerator struct {
	protoPackage string // proto 包名
	serviceName  string // 服务名
}

// NewProtoGenerator 创建Proto生成器
func NewProtoGenerator(protoPackage, serviceName string) *ProtoGenerator {
	return &ProtoGenerator{
		protoPackage: protoPackage,
		serviceName:  serviceName,
	}
}

// GenerateRequest 生成请求消息
func (g *ProtoGenerator) GenerateRequest(table *TableInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("message %sRequest {\n", upperFirst(table.TableName)))
	sb.WriteString("    // 请求ID\n")
	sb.WriteString("    string request_id = 1;\n")

	// 添加查询参数字段
	for i, col := range table.Columns {
		if !col.IsPrimary {
			num := i + 2
			sb.WriteString(fmt.Sprintf("    // %s: %s\n", col.Name, col.Comment))
			sb.WriteString(fmt.Sprintf("    %s %s = %d;\n", col.ProtoType, lowerFirst(toCamelCase(col.Name)), num))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// GenerateResponse 生成响应消息
func (g *ProtoGenerator) GenerateResponse(table *TableInfo) string {
	var sb strings.Builder

	// 单条记录响应
	sb.WriteString(fmt.Sprintf("message %sResponse {\n", upperFirst(table.TableName)))
	sb.WriteString("    // 响应状态\n")
	sb.WriteString("    int32 code = 1;\n")
	sb.WriteString("    string message = 2;\n")
	sb.WriteString(fmt.Sprintf("    %s data = 3;\n", upperFirst(table.TableName)))
	sb.WriteString("}\n\n")

	// 列表响应
	sb.WriteString(fmt.Sprintf("message %sListResponse {\n", upperFirst(table.TableName)))
	sb.WriteString("    int32 code = 1;\n")
	sb.WriteString("    string message = 2;\n")
	sb.WriteString(fmt.Sprintf("    repeated %s data_list = 3;\n", upperFirst(table.TableName)))
	sb.WriteString("    int32 total = 4;\n")
	sb.WriteString("}\n\n")

	// 分页响应
	sb.WriteString(fmt.Sprintf("message %sPageResponse {\n", upperFirst(table.TableName)))
	sb.WriteString("    int32 code = 1;\n")
	sb.WriteString("    string message = 2;\n")
	sb.WriteString(fmt.Sprintf("    repeated %s data_list = 3;\n", upperFirst(table.TableName)))
	sb.WriteString("    int64 total = 4;\n")
	sb.WriteString("    int32 page = 5;\n")
	sb.WriteString("    int32 page_size = 6;\n")
	sb.WriteString("}\n")

	return sb.String()
}

// GenerateDataMessage 生成数据消息体
func (g *ProtoGenerator) GenerateDataMessage(table *TableInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("message %s {\n", upperFirst(table.TableName)))
	sb.WriteString(fmt.Sprintf("    // %s\n", table.Comment))

	for i, col := range table.Columns {
		num := i + 1
		nullable := ""
		if col.IsNullable {
			nullable = "optional "
		}
		sb.WriteString(fmt.Sprintf("    %s%s %s = %d; // %s\n",
			nullable, col.ProtoType, lowerFirst(toCamelCase(col.Name)), num, col.Comment))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// GenerateService 生成服务定义
func (g *ProtoGenerator) GenerateService(table *TableInfo) string {
	var sb strings.Builder

	upperName := upperFirst(table.TableName)
	lowerName := lowerFirst(table.TableName)

	sb.WriteString(fmt.Sprintf("// %sService %s服务\n", upperName, table.Comment))
	sb.WriteString(fmt.Sprintf("service %sService {\n", upperName))
	sb.WriteString(fmt.Sprintf("    // Create 创建%s\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc Create%s(%sRequest) returns (%sResponse);\n", upperName, upperName, upperName))
	sb.WriteString(fmt.Sprintf("    // Update 更新%s\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc Update%s(%sRequest) returns (%sResponse);\n", upperName, upperName, upperName))
	sb.WriteString(fmt.Sprintf("    // Delete 删除%s\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc Delete%s(%sRequest) returns (%sResponse);\n", upperName, upperName, upperName))
	sb.WriteString(fmt.Sprintf("    // GetByID 根据ID获取%s\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc Get%sByID(%sRequest) returns (%sResponse);\n", upperName, upperName, upperName))
	sb.WriteString(fmt.Sprintf("    // List 获取%s列表\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc List%s(%sRequest) returns (%sListResponse);\n", upperName, upperName, upperName))
	sb.WriteString(fmt.Sprintf("    // Page 分页获取%s\n", table.Comment))
	sb.WriteString(fmt.Sprintf("    rpc Page%s(%sRequest) returns (%sPageResponse);\n", upperName, upperName, upperName))
	sb.WriteString("}\n")

	_ = lowerName // 避免未使用
	return sb.String()
}

// GenerateProtoFile 生成完整的proto文件
func (g *ProtoGenerator) GenerateProtoFile(table *TableInfo) string {
	var sb strings.Builder

	// 文件头
	sb.WriteString(fmt.Sprintf(`// Code generated by gospacex. DO NOT EDIT.
// version: 1.0.0
// generated at: %s
// table: %s

syntax = "proto3";

package %s;

option go_package = "github.com/example/%s/%s";

import "google/protobuf/timestamp.proto";

`,
		time.Now().Format("2006-01-02 15:04:05"),
		table.TableName,
		g.protoPackage,
		g.protoPackage,
		table.TableName))

	// 数据消息体
	sb.WriteString(g.GenerateDataMessage(table))
	sb.WriteString("\n")

	// 请求消息
	sb.WriteString(g.GenerateRequest(table))
	sb.WriteString("\n")

	// 响应消息
	sb.WriteString(g.GenerateResponse(table))
	sb.WriteString("\n")

	// 服务定义
	sb.WriteString(g.GenerateService(table))

	return sb.String()
}

// GenerateAllProto 生成多个表的proto文件
func (g *ProtoGenerator) GenerateAllProto(tables []*TableInfo) map[string]string {
	result := make(map[string]string)
	for _, table := range tables {
		result[table.TableName+".proto"] = g.GenerateProtoFile(table)
	}
	return result
}

// 类型映射
var typeMapping = map[string]string{
	// MySQL 类型映射
	"bigint":      "int64",
	"int":         "int32",
	"integer":     "int32",
	"smallint":    "int32",
	"tinyint":     "int32",
	"mediumint":   "int32",
	"float":       "float",
	"double":      "double",
	"decimal":     "double",
	"dec":         "double",
	"numeric":     "double",
	"varchar":     "string",
	"char":        "string",
	"text":        "string",
	"mediumtext":  "string",
	"longtext":    "string",
	"tinytext":    "string",
	"datetime":    "int64",
	"date":        "int64",
	"time":        "int64",
	"timestamp":   "int64",
	"blob":        "bytes",
	"tinyblob":    "bytes",
	"mediumblob":  "bytes",
	"longblob":    "bytes",
	"bit":         "bool",
	"boolean":     "bool",
	"bool":        "bool",
	"json":        "string",
	"enum":        "string",
	"set":         "string",
}

// Go 类型映射
var goTypeMapping = map[string]string{
	"bigint":      "int64",
	"int":         "int",
	"integer":     "int",
	"smallint":    "int32",
	"tinyint":     "int32",
	"mediumint":   "int32",
	"float":       "float32",
	"double":      "float64",
	"decimal":     "float64",
	"dec":         "float64",
	"numeric":     "float64",
	"varchar":     "string",
	"char":        "string",
	"text":        "string",
	"mediumtext":  "string",
	"longtext":    "string",
	"tinytext":    "string",
	"datetime":    "time.Time",
	"date":        "time.Time",
	"time":        "time.Time",
	"timestamp":   "time.Time",
	"blob":        "[]byte",
	"tinyblob":    "[]byte",
	"mediumblob":  "[]byte",
	"longblob":    "[]byte",
	"bit":         "bool",
	"boolean":     "bool",
	"bool":        "bool",
	"json":        "string",
	"enum":        "string",
	"set":         "string",
}

// GetProtoType 获取Proto类型
func GetProtoType(dbType string) string {
	dbType = strings.ToLower(dbType)
	dbType = strings.Split(dbType, "(")[0] // 移除长度参数
	dbType = strings.Split(dbType, " ")[0]

	if protoType, ok := typeMapping[dbType]; ok {
		return protoType
	}
	return "string" // 默认string类型
}

// GetGoType 获取Go类型
func GetGoType(dbType string) string {
	dbType = strings.ToLower(dbType)
	dbType = strings.Split(dbType, "(")[0]
	dbType = strings.Split(dbType, " ")[0]

	if goType, ok := goTypeMapping[dbType]; ok {
		return goType
	}
	return "string"
}

// toCamelCase 转换为驼峰命名
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// upperFirst 首字母大写
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// lowerFirst 首字母小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
