package generator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLToThriftGenerator converts MySQL tables to Thrift IDL
type MySQLToThriftGenerator struct {
	dsn       string
	tableName string
	namespace string
	outputDir string
}

// NewMySQLToThriftGenerator creates a new MySQL to Thrift generator
func NewMySQLToThriftGenerator(dsn, tableName, namespace, outputDir string) *MySQLToThriftGenerator {
	return &MySQLToThriftGenerator{
		dsn:       dsn,
		tableName: tableName,
		namespace: namespace,
		outputDir: outputDir,
	}
}

// TableInfo stores MySQL table schema information
type TableInfo struct {
	TableName  string
	Columns    []ColumnInfo
	PrimaryKey string
}

// ColumnInfo stores MySQL column information
type ColumnInfo struct {
	Name       string
	Type       string
	Nullable   bool
	Comment    string
	ThriftType string
}

// Generate generates Thrift IDL from MySQL table
func (g *MySQLToThriftGenerator) Generate() error {
	db, err := sql.Open("mysql", g.dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	tableInfo, err := g.getTableSchema(db)
	if err != nil {
		return fmt.Errorf("get table schema: %w", err)
	}

	if err := g.mapTypes(tableInfo); err != nil {
		return fmt.Errorf("map types: %w", err)
	}

	thriftContent, err := g.renderThriftIDL(tableInfo)
	if err != nil {
		return fmt.Errorf("render thrift IDL: %w", err)
	}

	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	thriftFile := filepath.Join(g.outputDir, g.tableName+".thrift")
	if err := os.WriteFile(thriftFile, []byte(thriftContent), 0644); err != nil {
		return fmt.Errorf("write thrift file: %w", err)
	}

	return nil
}

func (g *MySQLToThriftGenerator) getTableSchema(db *sql.DB) (*TableInfo, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COLUMN_COMMENT,
			COLUMN_KEY
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := db.Query(query, g.tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableInfo := &TableInfo{
		TableName: g.tableName,
		Columns:   []ColumnInfo{},
	}

	for rows.Next() {
		var col ColumnInfo
		var columnKey sql.NullString

		err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Comment, &columnKey)
		if err != nil {
			return nil, err
		}

		if columnKey.Valid && columnKey.String == "PRI" {
			tableInfo.PrimaryKey = col.Name
		}

		tableInfo.Columns = append(tableInfo.Columns, col)
	}

	if len(tableInfo.Columns) == 0 {
		return nil, fmt.Errorf("table %s not found", g.tableName)
	}

	return tableInfo, nil
}

func (g *MySQLToThriftGenerator) mapTypes(tableInfo *TableInfo) error {
	for i, col := range tableInfo.Columns {
		thriftType, err := g.mysqlToThriftType(col.Type, col.Nullable)
		if err != nil {
			return fmt.Errorf("column %s: %w", col.Name, err)
		}
		tableInfo.Columns[i].ThriftType = thriftType
	}
	return nil
}

func (g *MySQLToThriftGenerator) mysqlToThriftType(mysqlType string, nullable bool) (string, error) {
	mysqlType = strings.ToLower(mysqlType)

	switch {
	case strings.HasPrefix(mysqlType, "tinyint"),
		strings.HasPrefix(mysqlType, "smallint"),
		strings.HasPrefix(mysqlType, "mediumint"),
		strings.HasPrefix(mysqlType, "int"),
		strings.HasPrefix(mysqlType, "year"):
		if nullable {
			return "optional i32", nil
		}
		return "i32", nil

	case strings.HasPrefix(mysqlType, "bigint"):
		if nullable {
			return "optional i64", nil
		}
		return "i64", nil

	case strings.HasPrefix(mysqlType, "float"),
		strings.HasPrefix(mysqlType, "double"),
		strings.HasPrefix(mysqlType, "decimal"):
		if nullable {
			return "optional double", nil
		}
		return "double", nil

	case strings.HasPrefix(mysqlType, "char"),
		strings.HasPrefix(mysqlType, "varchar"),
		strings.HasPrefix(mysqlType, "text"),
		strings.HasPrefix(mysqlType, "enum"),
		strings.HasPrefix(mysqlType, "set"):
		if nullable {
			return "optional string", nil
		}
		return "string", nil

	case strings.HasPrefix(mysqlType, "tinyblob"),
		strings.HasPrefix(mysqlType, "blob"),
		strings.HasPrefix(mysqlType, "mediumblob"),
		strings.HasPrefix(mysqlType, "longblob"),
		strings.HasPrefix(mysqlType, "binary"),
		strings.HasPrefix(mysqlType, "varbinary"):
		return "binary", nil

	case strings.HasPrefix(mysqlType, "date"),
		strings.HasPrefix(mysqlType, "datetime"),
		strings.HasPrefix(mysqlType, "timestamp"):
		if nullable {
			return "optional i64", nil
		}
		return "i64", nil

	case strings.HasPrefix(mysqlType, "time"):
		if nullable {
			return "optional i32", nil
		}
		return "i32", nil

	case strings.HasPrefix(mysqlType, "json"):
		if nullable {
			return "optional string", nil
		}
		return "string", nil

	default:
		return "", fmt.Errorf("unsupported MySQL type: %s", mysqlType)
	}
}

func (g *MySQLToThriftGenerator) renderThriftIDL(tableInfo *TableInfo) (string, error) {
	tmpl, err := template.New("thrift").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
		"eq":         func(a, b interface{}) bool { return a == b },
		"ne":         func(a, b interface{}) bool { return a != b },
		"index": func(cols []ColumnInfo, i int) interface{} {
			if i >= 0 && i < len(cols) {
				return cols[i]
			}
			return nil
		},
	}).Parse(thriftTemplate)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	err = tmpl.Execute(&output, map[string]interface{}{
		"TableName":  tableInfo.TableName,
		"Namespace":  g.namespace,
		"Columns":    tableInfo.Columns,
		"PrimaryKey": tableInfo.PrimaryKey,
	})
	return output.String(), err
}

const thriftTemplate = `// Auto-generated by gospacex from MySQL table: {{ .TableName }}

namespace go {{ .Namespace }}

struct {{ .TableName }} {
{{- range .Columns }}
    {{- if eq .Name $.PrimaryKey }}
    1: {{ .ThriftType }} {{ .Name }}, // Primary key
    {{- else }}
    {{ .ThriftType }} {{ .Name }}, // {{ .Comment }}
    {{- end }}
{{- end }}
}

// Request/Response structures for CRUD operations

struct Get{{ .TableName }}Req {
    1: required {{ if eq (index .Columns 0).ThriftType "i64" }}i64{{ else }}string{{ end }} id
}

struct Get{{ .TableName }}Resp {
    1: {{ .TableName }} data
}

struct Create{{ .TableName }}Req {
{{- range $i, $col := .Columns }}
{{- if ne $col.Name $.PrimaryKey }}
    {{ if $col.Nullable }}optional {{ end }}{{ trimPrefix $col.ThriftType "optional " }} {{ $col.Name }},
{{- end }}
{{- end }}
}

struct Create{{ .TableName }}Resp {
    1: {{ if eq (index .Columns 0).ThriftType "i64" }}i64{{ else }}string{{ end }} id
}

struct Update{{ .TableName }}Req {
    1: required {{ if eq (index .Columns 0).ThriftType "i64" }}i64{{ else }}string{{ end }} id
{{- range $i, $col := .Columns }}
{{- if ne $col.Name $.PrimaryKey }}
    {{ if $col.Nullable }}optional {{ end }}{{ trimPrefix $col.ThriftType "optional " }} {{ $col.Name }},
{{- end }}
{{- end }}
}

struct Update{{ .TableName }}Resp {
    1: bool success
}

struct Delete{{ .TableName }}Req {
    1: required {{ if eq (index .Columns 0).ThriftType "i64" }}i64{{ else }}string{{ end }} id
}

struct Delete{{ .TableName }}Resp {
    1: bool success
}

struct List{{ .TableName }}Req {
    1: optional i32 page,
    2: optional i32 size,
}

struct List{{ .TableName }}Resp {
    1: list<{{ .TableName }}> data,
    2: i64 total,
}

service {{ .TableName }}Service {
    Get{{ .TableName }}Resp get{{ .TableName }}(1: Get{{ .TableName }}Req req),
    Create{{ .TableName }}Resp create{{ .TableName }}(1: Create{{ .TableName }}Req req),
    Update{{ .TableName }}Resp update{{ .TableName }}(1: Update{{ .TableName }}Req req),
    Delete{{ .TableName }}Resp delete{{ .TableName }}(1: Delete{{ .TableName }}Req req),
    List{{ .TableName }}Resp list{{ .TableName }}(1: List{{ .TableName }}Req req),
}
`
