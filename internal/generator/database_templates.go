package generator

import (
	"strings"
)

// DatabaseTemplates 数据库模板
type DatabaseTemplates struct{}

// GenerateMySQLConfig 生成 MySQL 配置
func (dt *DatabaseTemplates) GenerateMySQLConfig() string {
	return `# MySQL 配置
database:
  mysql:
    driver: mysql
    dsn: "user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true&loc=Local"
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: 3600  # seconds
`
}

// GeneratePostgresConfig 生成 PostgreSQL 配置
func (dt *DatabaseTemplates) GeneratePostgresConfig() string {
	return `# PostgreSQL 配置
database:
  postgres:
    driver: postgres
    dsn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: 3600
`
}

// GenerateRedisConfig 生成 Redis 配置
func (dt *DatabaseTemplates) GenerateRedisConfig() string {
	return `# Redis 配置
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 5
`
}

// GenerateElasticsearchConfig 生成 Elasticsearch 配置
func (dt *DatabaseTemplates) GenerateElasticsearchConfig() string {
	return `# Elasticsearch 配置
elasticsearch:
  addresses:
    - "http://localhost:9200"
  username: ""
  password: ""
  sniff: false
  healthcheck_interval: 60
`
}

// GenerateMongoDBConfig 生成 MongoDB 配置
func (dt *DatabaseTemplates) GenerateMongoDBConfig() string {
	return `# MongoDB 配置
mongodb:
  uri: "mongodb://localhost:27017"
  database: "mydb"
  max_pool_size: 100
  min_pool_size: 10
`
}

// GenerateDBConfig 根据数据库类型生成配置
func (dt *DatabaseTemplates) GenerateDBConfig(dbTypes []string) string {
	var configs []string

	for _, dbType := range dbTypes {
		switch strings.ToLower(dbType) {
		case "mysql":
			configs = append(configs, dt.GenerateMySQLConfig())
		case "postgresql", "postgres", "pg":
			configs = append(configs, dt.GeneratePostgresConfig())
		case "redis":
			configs = append(configs, dt.GenerateRedisConfig())
		case "elasticsearch", "es":
			configs = append(configs, dt.GenerateElasticsearchConfig())
		case "mongodb", "mongo":
			configs = append(configs, dt.GenerateMongoDBConfig())
		}
	}

	return strings.Join(configs, "\n")
}
