package generator

import (
	
	"os"
	"path/filepath"
)

// DatabaseIntegrationGenerator 数据库集成生成器
type DatabaseIntegrationGenerator struct {
	outputDir string
	databases []string
}

// NewDatabaseIntegrationGenerator creates new database integration generator
func NewDatabaseIntegrationGenerator(outputDir string, databases []string) *DatabaseIntegrationGenerator {
	return &DatabaseIntegrationGenerator{
		outputDir: outputDir,
		databases: databases,
	}
}

// Generate generates database integration code
func (g *DatabaseIntegrationGenerator) Generate() error {
	dirs := []string{
		"internal/dal/mysql",
		"internal/dal/postgres",
		"internal/dal/redis",
		"internal/dal/elasticsearch",
		"internal/dal/mongodb",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"internal/dal/mysql/init.go":        g.mysqlInitContent(),
		"internal/dal/postgres/init.go":     g.postgresInitContent(),
		"internal/dal/redis/init.go":        g.redisInitContent(),
		"internal/dal/elasticsearch/init.go": g.esInitContent(),
		"internal/dal/mongodb/init.go":      g.mongoInitContent(),
		"internal/dal/factory.go":           g.factoryContent(),
		"configs/database.yaml":             g.databaseConfigContent(),
	}

	for path, content := range files {
		fullPath := filepath.Join(g.outputDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (g *DatabaseIntegrationGenerator) mysqlInitContent() string {
	return `package mysql

import (
	
	"os"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init initializes MySQL connection
func Init() {
	dsn := fmt.Sprintf("%%s:%%s@tcp(%%s:3306)/%%s?parseTime=true",
		getEnv("MYSQL_USER", "root"),
		getEnv("MYSQL_PASSWORD", ""),
		getEnv("MYSQL_HOST", "localhost"),
		getEnv("MYSQL_DATABASE", "mydb"))
	
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil { panic(err) }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *DatabaseIntegrationGenerator) postgresInitContent() string {
	return `package postgres

import (
	
	"os"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init initializes PostgreSQL connection
func Init() {
	dsn := fmt.Sprintf("host=%%s user=%%s password=%%s dbname=%%s port=5432",
		getEnv("PG_HOST", "localhost"),
		getEnv("PG_USER", "postgres"),
		getEnv("PG_PASSWORD", ""),
		getEnv("PG_DATABASE", "mydb"))
	
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { panic(err) }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *DatabaseIntegrationGenerator) redisInitContent() string {
	return `package redis

import (
	"context"
	
	"os"
	"time"
	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// Init initializes Redis connection
func Init() {
	Client = redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := Client.Ping(ctx).Err(); err != nil { panic(err) }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

// Close closes connection
func Close() {
	if Client != nil { Client.Close() }
}
`
}

func (g *DatabaseIntegrationGenerator) esInitContent() string {
	return `package elasticsearch

import (
	"github.com/elastic/go-elasticsearch/v8"
	"os"
)

var Client *elasticsearch.Client

// Init initializes Elasticsearch connection
func Init() {
	cfg := elasticsearch.Config{
		Addresses: []string{getEnv("ES_ADDR", "http://localhost:9200")},
		Username:  os.Getenv("ES_USER"),
		Password:  os.Getenv("ES_PASSWORD"),
	}
	
	var err error
	Client, err = elasticsearch.NewClient(cfg)
	if err != nil { panic(err) }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *DatabaseIntegrationGenerator) mongoInitContent() string {
	return `package mongodb

import (
	"context"
	
	"os"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var Database *mongo.Database

// Init initializes MongoDB connection
func Init() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	
	uri := getEnv("MONGO_URI", "mongodb://localhost:27017")
	Client, _ = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	
	dbName := getEnv("MONGO_DB", "mydb")
	Database = Client.Database(dbName)
	
	if err := Client.Ping(ctx, nil); err != nil { panic(err) }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

// Close closes connection
func Close() {
	if Client != nil {
		Client.Disconnect(context.Background())
	}
}
`
}

func (g *DatabaseIntegrationGenerator) factoryContent() string {
	return `package dal

import (
	"%s/internal/dal/mysql"
	"%s/internal/dal/redis"
)

// Init initializes all databases
func Init() {
	mysql.Init()
	redis.Init()
}

// Close closes all connections
func Close() {
	redis.Close()
}
`
}

func (g *DatabaseIntegrationGenerator) databaseConfigContent() string {
	return `# Database Configuration

mysql:
  host: localhost
  port: 3306
  user: root
  password: ${MYSQL_PASSWORD}
  database: mydb

postgres:
  host: localhost
  port: 5432
  user: postgres
  password: ${PG_PASSWORD}
  database: mydb

redis:
  addr: localhost:6379
  password: ${REDIS_PASSWORD}
  db: 0

elasticsearch:
  addr: http://localhost:9200
  user: ${ES_USER}
  password: ${ES_PASSWORD}

mongodb:
  uri: mongodb://localhost:27017
  database: mydb
`
}
