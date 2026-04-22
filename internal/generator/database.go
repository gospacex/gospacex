package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DatabaseGenerator 数据库代码生成器
type DatabaseGenerator struct {
	outputDir string
}

// NewDatabaseGenerator 创建数据库生成器
func NewDatabaseGenerator(outputDir string) *DatabaseGenerator {
	return &DatabaseGenerator{
		outputDir: outputDir,
	}
}

// Generate 生成数据库相关代码
func (dg *DatabaseGenerator) Generate(dbTypes []string) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Join(dg.outputDir, "internal/database"), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(dg.outputDir, "internal/repository"), 0o755); err != nil {
		return err
	}

	// 生成数据库连接
	if err := dg.generateDatabaseConnection(dbTypes); err != nil {
		return err
	}

	// 生成 CRUD 基础代码
	if err := dg.generateBaseRepository(); err != nil {
		return err
	}

	// 生成示例 CRUD
	if err := dg.generateExampleRepository(); err != nil {
		return err
	}

	return nil
}

// generateDatabaseConnection 生成数据库连接代码
func (dg *DatabaseGenerator) generateDatabaseConnection(dbTypes []string) error {
	var imports []string
	var connFields []string
	var initFuncs []string

	for _, dbType := range dbTypes {
		switch strings.ToLower(dbType) {
		case "mysql", "postgresql", "postgres", "pg":
			imports = append(imports, "\t\"gorm.io/gorm\"")
			connFields = append(connFields, "\tDB       *gorm.DB")
			initFuncs = append(initFuncs, dg.generateGORMInit(strings.ToLower(dbType)))
		case "redis":
			imports = append(imports, "\t\"github.com/redis/go-redis/v9\"")
			connFields = append(connFields, "\tRedis    *redis.Client")
			initFuncs = append(initFuncs, dg.generateRedisInit())
		case "elasticsearch", "es":
			imports = append(imports, "\t\"github.com/elastic/go-elasticsearch/v8\"")
			connFields = append(connFields, "\tElastic  *elasticsearch.Client")
			initFuncs = append(initFuncs, dg.generateElasticsearchInit())
		case "mongodb", "mongo":
			imports = append(imports, "\t\"go.mongodb.org/mongo-driver/mongo\"")
			connFields = append(connFields, "\tMongoDB  *mongo.Client")
			initFuncs = append(initFuncs, dg.generateMongoDBInit())
		}
	}

	content := dg.buildDatabaseConnectionFile(imports, connFields, initFuncs)

	return os.WriteFile(
		filepath.Join(dg.outputDir, "internal/database/connection.go"),
		[]byte(content),
		0o644,
	)
}

func (dg *DatabaseGenerator) generateGORMInit(dbType string) string {
	return fmt.Sprintf(`
// Init%s initializes %s database connection
func Init%s(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(%s.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect %s: %%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	log.Printf("%%s database connected", "%s")
	return db, nil
}`,
		strings.Title(dbType), dbType,
		strings.Title(dbType),
		strings.Title(dbType),
		dbType,
		dbType,
	)
}

func (dg *DatabaseGenerator) generateRedisInit() string {
	return `
// InitRedis initializes Redis connection
func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	log.Println("Redis connected")
	return client, nil
}`
}

func (dg *DatabaseGenerator) generateElasticsearchInit() string {
	return `
// InitElasticsearch initializes Elasticsearch connection
func InitElasticsearch(cfg config.ElasticsearchConfig) (*elasticsearch.Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("connect elasticsearch: %w", err)
	}

	res, err := es.Info()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	log.Println("Elasticsearch connected")
	return es, nil
}`
}

func (dg *DatabaseGenerator) generateMongoDBInit() string {
	return `
// InitMongoDB initializes MongoDB connection
func InitMongoDB(cfg config.MongoDBConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("MongoDB connected")
	return client, nil
}`
}

func (dg *DatabaseGenerator) buildDatabaseConnectionFile(imports, connFields, initFuncs []string) string {
	importsStr := strings.Join(imports, "\n")
	connFieldsStr := strings.Join(connFields, "\n")
	initFuncsStr := strings.Join(initFuncs, "\n")

	return fmt.Sprintf(`package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gospacex/gpx/internal/config"
%s
)

// Database 数据库连接管理器
type Database struct {
%s
}

// NewDatabase creates new database connection manager
func NewDatabase(cfg config.Config) (*Database, error) {
	db := &Database{}
	var err error

%s

	return db, err
}

// Close closes all database connections
func (d *Database) Close() error {
	if d.DB != nil {
		sqlDB, _ := d.DB.DB()
		sqlDB.Close()
	}
	if d.Redis != nil {
		d.Redis.Close()
	}
	if d.MongoDB != nil {
		d.MongoDB.Disconnect(context.Background())
	}
	return nil
}

%s
`, importsStr, connFieldsStr, dg.generateInitCalls(), initFuncsStr)
}

func (dg *DatabaseGenerator) generateInitCalls() string {
	return `	// Initialize connections based on config
	// Example:
	// db.DB, err = InitMySQL(cfg.Database.MySQL)
	// if err != nil {
	//     return nil, err
	// }
`
}

// generateBaseRepository 生成基础 CRUD 接口
func (dg *DatabaseGenerator) generateBaseRepository() error {
	content := `package repository

import "context"

// BaseRepository 基础 CRUD 接口
type BaseRepository[T any] interface {
	// Create 创建记录
	Create(ctx context.Context, entity *T) error

	// GetByID 根据 ID 查询
	GetByID(ctx context.Context, id interface{}) (*T, error)

	// List 列表查询
	List(ctx context.Context, offset, limit int) ([]T, error)

	// Update 更新记录
	Update(ctx context.Context, entity *T) error

	// Delete 删除记录
	Delete(ctx context.Context, id interface{}) error

	// Count 统计数量
	Count(ctx context.Context) (int64, error)
}
`
	return os.WriteFile(
		filepath.Join(dg.outputDir, "internal/repository/base.go"),
		[]byte(content),
		0o644,
	)
}

// generateExampleRepository 生成示例 CRUD 实现
func (dg *DatabaseGenerator) generateExampleRepository() error {
	content := `package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Example 示例实体
type Example struct {
	ID        uint           ` + "`gorm:\"primaryKey\"`" + `
	Name      string         ` + "`gorm:\"size:255;not null\"`" + `
	Email     string         ` + "`gorm:\"size:255;uniqueIndex\"`" + `
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt ` + "`gorm:\"index\"`" + `
}

// ExampleRepository Example 仓储
type ExampleRepository struct {
	db *gorm.DB
}

// NewExampleRepository creates new Example repository
func NewExampleRepository(db *gorm.DB) *ExampleRepository {
	return &ExampleRepository{db: db}
}

// Create creates a new example
func (r *ExampleRepository) Create(ctx context.Context, entity *Example) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// GetByID gets example by ID
func (r *ExampleRepository) GetByID(ctx context.Context, id interface{}) (*Example, error) {
	var entity Example
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// List lists examples
func (r *ExampleRepository) List(ctx context.Context, offset, limit int) ([]Example, error) {
	var entities []Example
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&entities).Error
	return entities, err
}

// Update updates an example
func (r *ExampleRepository) Update(ctx context.Context, entity *Example) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete deletes an example
func (r *ExampleRepository) Delete(ctx context.Context, id interface{}) error {
	return r.db.WithContext(ctx).Delete(&Example{}, id).Error
}

// Count counts examples
func (r *ExampleRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Example{}).Count(&count).Error
	return count, err
}
`
	return os.WriteFile(
		filepath.Join(dg.outputDir, "internal/repository/example.go"),
		[]byte(content),
		0o644,
	)
}
