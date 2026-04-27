package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"
)

// TemplateFieldData 模板字段数据
type TemplateFieldData struct {
	Name            string // 原始字段名 (snake_case, 如 image_input)
	ProtoGoName     string // proto 生成的 Go 字段名 (CamelCase, 如 ImageInput)
	GoType          string // Go 类型 (int64, string, float64, etc.)
	ProtoType       string // proto 类型 (int64, string, double, etc.)
	IsPrimary       bool
	IsNullable      bool
	Comment         string
	ZeroValue       string // Go 零值字符串表示 (如 "", 0, 0.0)
	GormTag         string // GORM 标签 (如 gorm:"column:id;primaryKey")
	ModelToRespCode string // model 字段转 proto 响应字段的代码 (如 int64(m.Id), m.Title)
}

// TemplateData 模板渲染数据
type TemplateData struct {
	AppName     string
	Module      string // 服务 module 名（如 "srvProduct"），用于 import 路径
	UpperModule string
	LowerModule string
	EntityName     string // entity 名（如 "Attr"），用于 struct/文件名
	UpperEntityName string // entity 名的 CamelCase（如 "ProductAttr"），用于 proto 类型引用
	LowerEntityName string // entity 名的首字母小写形式（如 "productAttr"），用于变量名
	BFFName     string
	TableName   string
	SrvPort     int    // 微服务 gRPC 端口
	Register    string // 注册中心类型: consul|etcd|""
	SrvDirName  string // 小驼峰 srv 目录名，如 srvProduct
	BffDirName  string // 小驼峰 bff 目录名，如 bffApi
	HasSoftDelete  bool   // 表是否有软删除字段（is_del 等）
	SoftDeleteCol  string // 软删除字段的列名，如 is_del、deleted、deleted_at
	HasCreatedAt   bool   // 表是否有 created_at/create_time 列
	HasUpdatedAt   bool   // 表是否有 updated_at/update_time 列
	PKGoType    string
	PKProtoName string // proto 中的主键字段名 (snake_case, 如 id)
	PKProtoGoName string // proto 生成的 Go 主键字段名 (CamelCase, 如 Id)
	PKModelConvert  string // 主键从 proto 到 model 的转换代码
	PKReqConvert    string // Get 请求中主键的转换
	PKUpdateReqConvert string // Update 请求中主键的转换
	PKDeleteReqConvert string // Delete 请求中主键的转换
	PKRespConvert   string // Create 响应中主键的转换
	RepoInitCode    string // repository 初始化代码
	Columns     []TemplateFieldData
	CreateFields []TemplateFieldData // Create 请求字段（排除自增主键）
	UpdateFields []TemplateFieldData // Update 请求字段（排除主键）
	HandlerRegs  string // main.go 中注册多个 handler 的代码
	ProtoImports []string // main.go 中多表场景下的 proto import 列表
}

// buildTemplateData 从 ColumnInfo 构建模板数据
// entityName 用于 struct 命名（可为空，此时等于 module）
func buildTemplateData(module string, columns []ColumnInfo, bffName string, tableName string, srvPort int, entityNames ...string) TemplateData {
	pkField := getPrimaryKeyField(columns)
	pkGoType := protoTypeToGoType(pkField.ProtoType)
	pkProtoGoName := toProtoGoFieldName(pkField.Name)
	upperModule := strings.ToUpper(module[:1]) + module[1:]
	lowerModule := strings.ToLower(module)
	// entityName 用于 struct 命名（若未提供则用 module）
	entityName := module
	if len(entityNames) > 0 && entityNames[0] != "" {
		entityName = entityNames[0]
	}
	upperEntityName := toProtoGoFieldName(entityName)
	lowerEntityName := strings.ToLower(upperEntityName[:1]) + upperEntityName[1:]

	var cols []TemplateFieldData
	var createFields []TemplateFieldData
	var updateFields []TemplateFieldData
	hasSoftDelete := false
	softDeleteCol := ""
	hasCreatedAt := false
	hasUpdatedAt := false

	for _, col := range columns {
		// 检测软删除字段
		if isSoftDeleteField(col.Name) {
			hasSoftDelete = true
			softDeleteCol = col.Name
		}
		// 检测时间戳字段（表中真实存在）
		lower := strings.ToLower(col.Name)
		if lower == "created_at" || lower == "createdat" || lower == "create_time" || lower == "createtime" {
			hasCreatedAt = true
		}
		if lower == "updated_at" || lower == "updatedat" || lower == "update_time" || lower == "updatetime" {
			hasUpdatedAt = true
		}
		fd := TemplateFieldData{
			Name:        col.Name,
			ProtoGoName: toProtoGoFieldName(col.Name),
			GoType:      protoTypeToGoType(col.ProtoType),
			ProtoType:   col.ProtoType,
			IsPrimary:   col.IsPrimary,
			IsNullable:      col.IsNullable,
			Comment:         col.Comment,
			ZeroValue:       goZeroValue(col.ProtoType),
			GormTag:         getGormTag(col),
		}
		// ModelToRespCode 需要在 fd 完全构建后生成（时间戳字段需要格式化）
		fd.ModelToRespCode = modelToRespFieldFromFD(fd)
		cols = append(cols, fd)

		// Create: 排除自增主键、时间戳字段、软删除字段
		if !(col.IsPrimary && strings.Contains(strings.ToLower(col.Type), "int")) && !isTimestampField(col.Name) && !isSoftDeleteField(col.Name) {
			createFields = append(createFields, fd)
		}
		// Update: 排除主键、时间戳字段、软删除字段
		if !col.IsPrimary && !isTimestampField(col.Name) && !isSoftDeleteField(col.Name) {
			updateFields = append(updateFields, fd)
		}
	}

	// 主键转换代码
	pkModelConvert := pkModelConvertCode(pkField)
	pkReqConvert := fmt.Sprintf("%s(req.%s)", pkGoType, pkProtoGoName)
	pkUpdateReqConvert := fmt.Sprintf("%s(req.%s)", pkGoType, pkProtoGoName)
	pkDeleteReqConvert := fmt.Sprintf("%s(req.%s)", pkGoType, pkProtoGoName)
	pkRespConvert := fmt.Sprintf("%s(m.%s)", pkGoType, toProtoGoFieldName(pkField.Name))
	repoInitCode := "repository.New" + upperEntityName + "Repo(db)"

	return TemplateData{
		AppName:            microAppName,
		Module:             module,
		UpperModule:        upperModule,
		LowerModule:        lowerModule,
		EntityName:         entityName,
		UpperEntityName:    upperEntityName,
		LowerEntityName:    lowerEntityName,
		BFFName:            bffName,
		TableName:          tableName,
		SrvPort:           srvPort,
		Register:           microAppRegister,
		SrvDirName:         toSrvDirName(module),
		BffDirName:         toBffDirName(bffName),
		HasSoftDelete:      hasSoftDelete,
		SoftDeleteCol:      softDeleteCol,
		HasCreatedAt:       hasCreatedAt,
		HasUpdatedAt:       hasUpdatedAt,
		PKGoType:           pkGoType,
		PKProtoName:        getLowerFirst(pkField.Name),
		PKProtoGoName:      pkProtoGoName,
		PKModelConvert:     pkModelConvert,
		PKReqConvert:       pkReqConvert,
		PKUpdateReqConvert: pkUpdateReqConvert,
		PKDeleteReqConvert: pkDeleteReqConvert,
		PKRespConvert:      pkRespConvert,
		RepoInitCode:       repoInitCode,
		Columns:            cols,
		CreateFields:       createFields,
		UpdateFields:       updateFields,
	}
}

// goZeroValue 返回 Go 类型的零值字符串
func goZeroValue(protoType string) string {
	switch protoType {
	case "int64", "int32":
		return "0"
	case "double":
		return "0.0"
	case "bool":
		return "false"
	default:
		return `""`
	}
}

// pkModelConvertCode 生成主键从 proto 类型到 model 类型的转换代码
func pkModelConvertCode(pkField ColumnInfo) string {
	// service 和 repo 统一使用 proto 的 Go 类型（int64），无需转换
	return "id"
}

// renderTemplate 渲染模板文件并写入目标路径
func renderTemplate(tmplPath string, data TemplateData, outputPath string) error {
	// 读取模板文件
	tmplContent, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("读取模板失败 %s: %w", tmplPath, err)
	}

	// 创建模板，注册自定义函数
	tmpl := template.New(filepath.Base(tmplPath)).Funcs(template.FuncMap{
		"toProtoGoFieldName": toProtoGoFieldName,
		"protoTypeToGoType":  protoTypeToGoType,
		"gormTag": func(col ColumnInfo) string {
			return getGormTag(col)
		},
		"modelToRespField": func(fd TemplateFieldData) string {
			return modelToRespFieldFromFD(fd)
		},
	})

	tmpl, err = tmpl.Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("解析模板失败 %s: %w", tmplPath, err)
	}

	// 确保输出目录存在
	os.MkdirAll(filepath.Dir(outputPath), 0755)

	// 创建输出文件
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败 %s: %w", outputPath, err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

// modelToRespCode 根据字段类型生成 model -> proto 响应字段的转换代码
func modelToRespCode(col ColumnInfo) string {
	protoGoName := toProtoGoFieldName(col.Name)
	switch col.ProtoType {
	case "int64":
		return fmt.Sprintf("int64(m.%s)", protoGoName)
	case "int32":
		return fmt.Sprintf("int32(m.%s)", protoGoName)
	case "double":
		return fmt.Sprintf("float64(m.%s)", protoGoName)
	default:
		return fmt.Sprintf("m.%s", protoGoName)
	}
}

// modelToRespFieldFromFD 根据字段类型生成 model -> proto 响应字段的转换代码 (TemplateFieldData)
func modelToRespFieldFromFD(fd TemplateFieldData) string {
	modelName := fd.ProtoGoName
	// 时间戳字段：model 是 time.Time，proto 是 string，需要格式化转换
	if isTimestampField(fd.Name) {
		return fmt.Sprintf("m.%s.Format(\"2006-01-02 15:04:05\")", modelName)
	}
	switch fd.ProtoType {
	case "int64":
		return fmt.Sprintf("int64(m.%s)", modelName)
	case "int32":
		return fmt.Sprintf("int32(m.%s)", modelName)
	case "double":
		return fmt.Sprintf("float64(m.%s)", modelName)
	default:
		return fmt.Sprintf("m.%s", modelName)
	}
}

// isTimestampField 判断是否为时间戳字段
func isTimestampField(name string) bool {
	lower := strings.ToLower(name)
	return lower == "created_at" || lower == "updated_at" ||
		lower == "createdat" || lower == "updatedat" ||
		lower == "create_time" || lower == "update_time" ||
		lower == "createtime" || lower == "updatetime"
}

// isSoftDeleteField 判断是否为软删除字段（is_del、deleted、is_deleted 等）
func isSoftDeleteField(name string) bool {
	lower := strings.ToLower(name)
	return lower == "is_del" || lower == "isdel" ||
		lower == "deleted" || lower == "is_deleted" || lower == "isdeleted" ||
		lower == "del_flag" || lower == "delflag"
}
