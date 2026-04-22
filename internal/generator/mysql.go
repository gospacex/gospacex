package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// MySQLGenerator generates MySQL database templates
type MySQLGenerator struct {
	OutputDir string
}

// NewMySQLGenerator creates a new MySQLGenerator
func NewMySQLGenerator(outputDir string) *MySQLGenerator {
	return &MySQLGenerator{
		OutputDir: outputDir,
	}
}

// Generate creates MySQL template files
func (g *MySQLGenerator) Generate(modelName string) error {
	files := []struct {
		path    string
		content string
	}{
		{"config.go.tmpl", mysqlConfigTemplate},
		{"client.go.tmpl", mysqlClientTemplate},
		{"crud.go.tmpl", mysqlCRUDTemplate},
	}

	dir := filepath.Join(g.OutputDir, "templates", "database", "mysql")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	for _, f := range files {
		tmpl, err := template.New(f.path).Parse(f.content)
		if err != nil {
			return fmt.Errorf("parse template %s: %w", f.path, err)
		}

		file, err := os.Create(filepath.Join(dir, f.path))
		if err != nil {
			return fmt.Errorf("create file %s: %w", f.path, err)
		}
		defer file.Close()

		data := map[string]string{
			"ModelName": modelName,
		}

		if err := tmpl.Execute(file, data); err != nil {
			return fmt.Errorf("execute template %s: %w", f.path, err)
		}
	}

	return nil
}

const mysqlConfigTemplate = `package mysql

import (
	"fmt"
)

// Config MySQL configuration
type Config struct {
	Host     string ` + "`yaml:\"host\" env:\"DB_HOST\" default:\"localhost\"`" + `
	Port     int    ` + "`yaml:\"port\" env:\"DB_PORT\" default:\"3306\"`" + `
	User     string ` + "`yaml:\"user\" env:\"DB_USER\"`" + `
	Password string ` + "`yaml:\"password\" env:\"DB_PASSWORD\"`" + `
	Database string ` + "`yaml:\"database\" env:\"DB_NAME\"`" + `
}

// DSN returns the MySQL DSN connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database)
}
`

const mysqlClientTemplate = `package mysql

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Client MySQL client wrapper
type Client struct {
	db *gorm.DB
}

// NewClient creates a new MySQL client
func NewClient(cfg *Config) (*Client, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Client{db: db}, nil
}

// DB returns the underlying gorm.DB
func (c *Client) DB() *gorm.DB {
	return c.db
}

// Close closes the database connection
func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck checks database connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
`

const mysqlCRUDTemplate = `package mysql

import (
	"context"

	"gorm.io/gorm"
)

// {{ .ModelName }}DAO Data Access Object for {{ .ModelName }}
type {{ .ModelName }}DAO struct {
	db *gorm.DB
}

// New{{ .ModelName }}DAO creates a new {{ .ModelName }}DAO
func New{{ .ModelName }}DAO(db *gorm.DB) *{{ .ModelName }}DAO {
	return &{{ .ModelName }}DAO{db: db}
}

// Create creates a new {{ .ModelName }} record
func (d *{{ .ModelName }}DAO) Create(ctx context.Context, m *{{ .ModelName }}) error {
	return d.db.WithContext(ctx).Create(m).Error
}

// GetByID gets a {{ .ModelName }} by ID
func (d *{{ .ModelName }}DAO) GetByID(ctx context.Context, id int64) (*{{ .ModelName }}, error) {
	var result {{ .ModelName }}
	err := d.db.WithContext(ctx).First(&result, id).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update updates a {{ .ModelName }} record
func (d *{{ .ModelName }}DAO) Update(ctx context.Context, m *{{ .ModelName }}) error {
	return d.db.WithContext(ctx).Save(m).Error
}

// Delete deletes a {{ .ModelName }} record by ID
func (d *{{ .ModelName }}DAO) Delete(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&{{ .ModelName }}{}, id).Error
}

// List lists {{ .ModelName }} records with pagination
func (d *{{ .ModelName }}DAO) List(ctx context.Context, offset, limit int) ([]*{{ .ModelName }}, error) {
	var results []*{{ .ModelName }}
	err := d.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&results).Error
	return results, err
}
`
