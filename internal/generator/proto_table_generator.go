package generator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ProtoField represents a database column
type ProtoField struct {
	Name       string // 字段名 (proto 风格, 蛇形转驼峰)
	ProtoType  string // Proto 类型 (string, int32, int64, bool, etc)
	Comment    string // 字段注释
	JSONTag    string // JSON 标签
	IsPrimary  bool   // 是否主键
	IsAutoIncr bool  // 是否自增
}

// ProtoTableInfo holds table metadata for proto generation
type ProtoTableInfo struct {
	TableName    string
	ServiceName  string // 首字母大写
	ModuleName   string // 模块名 (小写)
	PackageName  string
	ProjectName  string
	Fields       []ProtoField
	ReqStruct    string
	RespStruct   string
	CRUDMethods  []string
}

// MySQLTypeToProtoType maps MySQL types to Proto types
var MySQLTypeToProtoType = map[string]string{
	// 整数类型
	"tinyint":   "int32",
	"smallint":  "int32",
	"mediumint": "int32",
	"int":       "int32",
	"integer":   "int32",
	"bigint":    "int64",
	// 无符号整数
	"tinyint unsigned":   "int32",
	"smallint unsigned": "int32",
	"mediumint unsigned": "int32",
	"int unsigned":      "int64",
	"integer unsigned":  "int64",
	"bigint unsigned":   "int64",
	// 浮点类型
	"float":   "float",
	"double":  "double",
	"decimal": "double",
	"dec":     "double",
	// 字符串类型
	"char":       "string",
	"varchar":    "string",
	"tinytext":   "string",
	"text":       "string",
	"mediumtext": "string",
	"longtext":   "string",
	// 时间类型
	"date":      "string", // 使用 string 类型更灵活,格式为 RFC3339
	"datetime":  "string",
	"timestamp": "int64", // Unix 时间戳
	"time":      "string",
	"year":      "int32",
	// 二进制类型
	"tinyblob":   "bytes",
	"blob":       "bytes",
	"mediumblob": "bytes",
	"longblob":   "bytes",
	"binary":     "bytes",
	"varbinary":  "bytes",
	// 其他
	"bit":       "int64",
	"boolean":   "bool",
	"bool":      "bool",
	"json":      "string",
	"enum":      "string",
	"set":       "string",
}

// ProtoGenerator generates proto files from database tables
type ProtoGenerator struct {
	DB        *sql.DB
	OutputDir string
	ProjectName string
}

// NewProtoGenerator creates a new proto generator
func NewProtoGenerator(db *sql.DB, outputDir, projectName string) *ProtoGenerator {
	return &ProtoGenerator{
		DB:          db,
		OutputDir:   outputDir,
		ProjectName: projectName,
	}
}

// GenerateFromTable generates proto file from a database table
func (g *ProtoGenerator) GenerateFromTable(tableName string) (*ProtoTableInfo, error) {
	// 查询表结构
	rows, err := g.DB.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName))
	if err != nil {
		return nil, fmt.Errorf("query table columns: %w", err)
	}
	defer rows.Close()

	var fields []ProtoField
	var primaryKey string

	for rows.Next() {
		var field, ftype, null, key, extra, comment string
		var defaultVal, collation sql.NullString

		if err := rows.Scan(&field, &ftype, &collation, &null, &defaultVal, &key, &extra, &collation, &comment); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}

		// 跳过 gorm 软删除字段
		if field == "deleted_at" {
			continue
		}

		protoType := g.mysqlTypeToProtoType(ftype)
		protoName := snakeToCamel(strings.TrimPrefix(field, "is_")) // 去除 is_ 前缀
		isPrimary := key == "PRI"
		isAutoIncr := strings.Contains(strings.ToLower(extra), "auto_increment")
		// is_ 前缀的列映射为 bool 类型
		if strings.HasPrefix(field, "is_") {
			protoType = "bool"
		}

		if isPrimary {
			primaryKey = protoName
		}
		if isAutoIncr {
			_ = protoName // auto increment field (used for tracking)
		}

		fields = append(fields, ProtoField{
			Name:       protoName,
			ProtoType:  protoType,
			Comment:    comment,
			JSONTag:    fmt.Sprintf("%s", field),
			IsPrimary:  isPrimary,
			IsAutoIncr: isAutoIncr,
		})
	}

	// 获取服务名 (首字母大写) - 去除 eb_ 前缀
	strippedTable := strings.TrimPrefix(tableName, "eb_")
	serviceName := snakeToCamel(strippedTable)
	// 首字母大写
	serviceName = strings.ToUpper(serviceName[:1]) + serviceName[1:]
	moduleName := strings.ToLower(snakeToCamel(strippedTable))
	moduleName = strings.ToLower(moduleName[:1]) + moduleName[1:]

	info := &ProtoTableInfo{
		TableName:    tableName,
		ServiceName:  serviceName,
		ModuleName:   moduleName,
		PackageName:  serviceName,
		ProjectName:  g.ProjectName,
		Fields:       fields,
		ReqStruct:    g.generateReqStruct(serviceName),
		RespStruct:   g.generateRespStruct(serviceName),
		CRUDMethods:  g.generateCRUDMethods(serviceName, primaryKey),
	}

	return info, nil
}

// GenerateProtoFile generates the proto file from table info
func (g *ProtoGenerator) GenerateProtoFile(info *ProtoTableInfo, outPath string) error {
	tmpl := `syntax = "proto3";

package {{.PackageName}};

option go_package = "{{.ProjectName}}/common/kitex_gen/{{.ModuleName}}";

// {{.ServiceName}}Service - {{.TableName}} table
service {{.ServiceName}}Service {
  rpc Create(Create{{.ServiceName}}Req) returns (Create{{.ServiceName}}Resp);
  rpc Get(Get{{.ServiceName}}Req) returns (Get{{.ServiceName}}Resp);
  rpc List(List{{.ServiceName}}Req) returns (List{{.ServiceName}}Resp);
  rpc Update(Update{{.ServiceName}}Req) returns (Update{{.ServiceName}}Resp);
  rpc Delete(Delete{{.ServiceName}}Req) returns (Delete{{.ServiceName}}Resp);
}

{{.ReqStruct}}

{{.RespStruct}}
`

	t, err := template.New("proto").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	return t.Execute(file, info)
}

func (g *ProtoGenerator) mysqlTypeToProtoType(mysqlType string) string {
	// 提取基础类型 (去掉长度等参数)
	baseType := strings.ToLower(mysqlType)
	if idx := strings.Index(baseType, "("); idx > 0 {
		baseType = baseType[:idx]
	}
	baseType = strings.TrimSpace(baseType)

	if protoType, ok := MySQLTypeToProtoType[baseType]; ok {
		return protoType
	}

	// 默认使用 string
	return "string"
}

func (g *ProtoGenerator) generateReqStruct(serviceName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// Create%sReq - 创建请求\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Create%sReq {\n", serviceName))
	sb.WriteString("  // TODO: 添加创建所需字段\n")
	sb.WriteString("  string name = 1;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Get%sReq - 获取详情请求\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Get%sReq {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// List%sReq - 列表请求\n", serviceName))
	sb.WriteString(fmt.Sprintf("message List%sReq {\n", serviceName))
	sb.WriteString("  int32 page = 1;\n")
	sb.WriteString("  int32 page_size = 10;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Update%sReq - 更新请求\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Update%sReq {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("  // TODO: 添加更新字段\n")
	sb.WriteString("  string name = 2;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Delete%sReq - 删除请求\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Delete%sReq {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("}\n")

	return sb.String()
}

func (g *ProtoGenerator) generateRespStruct(serviceName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// Create%sResp - 创建响应\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Create%sResp {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("  bool success = 2;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Get%sResp - 获取详情响应\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Get%sResp {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("  // TODO: 添加响应字段\n")
	sb.WriteString("  string name = 2;\n")
	sb.WriteString("  int32 status = 3;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// List%sResp - 列表响应\n", serviceName))
	sb.WriteString(fmt.Sprintf("message List%sResp {\n", serviceName))
	sb.WriteString(fmt.Sprintf("  repeated %sItem items = 1;\n", serviceName))
	sb.WriteString("  int64 total = 2;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// %sItem - 列表项\n", serviceName))
	sb.WriteString(fmt.Sprintf("message %sItem {\n", serviceName))
	sb.WriteString("  int64 id = 1;\n")
	sb.WriteString("  // TODO: 添加列表字段\n")
	sb.WriteString("  string name = 2;\n")
	sb.WriteString("  int32 status = 3;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Update%sResp - 更新响应\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Update%sResp {\n", serviceName))
	sb.WriteString("  bool success = 1;\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// Delete%sResp - 删除响应\n", serviceName))
	sb.WriteString(fmt.Sprintf("message Delete%sResp {\n", serviceName))
	sb.WriteString("  bool success = 1;\n")
	sb.WriteString("}\n")

	return sb.String()
}

func (g *ProtoGenerator) generateCRUDMethods(serviceName, primaryKey string) []string {
	return []string{
		fmt.Sprintf("Create%s - 创建 %s", serviceName, serviceName),
		fmt.Sprintf("Get%s - 获取 %s 详情", serviceName, serviceName),
		fmt.Sprintf("List%s - 获取 %s 列表", serviceName, serviceName),
		fmt.Sprintf("Update%s - 更新 %s", serviceName, serviceName),
		fmt.Sprintf("Delete%s - 删除 %s", serviceName, serviceName),
	}
}
