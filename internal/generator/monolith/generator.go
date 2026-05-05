package monolith

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gospacex/gpx/internal/config"
	"github.com/gospacex/gpx/internal/template"
)

// Generator 单体应用生成器
type Generator struct {
	config *config.ProjectConfig
}

// NewGenerator 创建新的单体应用生成器
func NewGenerator(cfg *config.ProjectConfig) *Generator {
	return &Generator{
		config: cfg,
	}
}

// Generate 生成单体应用项目
func (g *Generator) Generate() error {
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return err
	}

	// 先根据配置生成可选组件
	if err := g.generateComponents(); err != nil {
		return fmt.Errorf("generate components: %w", err)
	}

	// 生成基础模板
	if err := g.generateBaseTemplates(); err != nil {
		return fmt.Errorf("generate base templates: %w", err)
	}

	return nil
}

// generateBaseTemplates 生成基础模板（必选组件）
func (g *Generator) generateBaseTemplates() error {
	baseDir := "templates/monolith"

	// 基础模板映射（必选）
	baseTemplates := map[string]string{
		"README.md":                   baseDir + "/README.md.tmpl",
		"cmd/main.go":                 baseDir + "/cmd/main.go.tmpl",
		"cmd/server/main.go":           baseDir + "/cmd/server/main.go.tmpl",
		"configs/config.yaml":          baseDir + "/configs/config.yaml.tmpl",
		"configs/server.yaml":          baseDir + "/configs/server.yaml.tmpl",
		"go.mod":                      baseDir + "/go.mod.tmpl",
		"internal/api/articleHandler.go": baseDir + "/internal/api/articleHandler.go.tmpl",
		"internal/handler/routes.go":  baseDir + "/internal/handler/routes.go.tmpl",
		"internal/shared/db.go":        baseDir + "/internal/shared/db.go.tmpl",
		"pkg/config/config.go":        baseDir + "/pkg/config/config.go.tmpl",
		"pkg/config/config_test.go":   baseDir + "/pkg/config/config_test.go.tmpl",
	}

	// 生成 deploy 目录（如果模板存在）
	if _, err := os.Stat(baseDir + "/deploy/docker-compose.yaml.tmpl"); err == nil {
		baseTemplates["deploy/docker-compose.yaml"] = baseDir + "/deploy/docker-compose.yaml.tmpl"
	}

	// 根据配置决定是否生成 examples
	if g.config.MQ != "" {
		baseTemplates["examples/consumer_example.go"] = baseDir + "/examples/consumer_example.go.tmpl"
		baseTemplates["examples/producer_example.go"] = baseDir + "/examples/producer_example.go.tmpl"
	}

	// 根据配置决定是否生成 internal/producer
	if g.config.MQ != "" {
		baseTemplates["internal/producer/kafka_producer.go"] = baseDir + "/internal/producer/kafka_producer.go.tmpl"
	}

	return g.copyTemplates(baseTemplates)
}

// generateComponents 根据配置生成额外组件（可选）
func (g *Generator) generateComponents() error {
	// 生成 Redis 组件
	if g.config.Cache == "redis" {
		if err := g.generateRedis(); err != nil {
			return fmt.Errorf("generate redis: %w", err)
		}
	}

	// 生成 Elasticsearch 组件
	if len(g.config.ES.Addresses) > 0 {
		if err := g.generateES(); err != nil {
			return fmt.Errorf("generate elasticsearch: %w", err)
		}
	}

	// 生成 MQ 组件
	if g.config.MQ != "" {
		if err := g.generateMQ(); err != nil {
			return fmt.Errorf("generate mq: %w", err)
		}
	}

	// 生成对象存储组件
	if g.config.ObjectDB != "" {
		if err := g.generateObjectStorage(); err != nil {
			return fmt.Errorf("generate object storage: %w", err)
		}
	}

	// 生成向量数据库组件
	if g.config.ZVecHost != "" || g.config.ZVecCollection != "" {
		if err := g.generateVector(); err != nil {
			return fmt.Errorf("generate vector: %w", err)
		}
	}

	return nil
}

// generateRedis 生成 Redis 组件
func (g *Generator) generateRedis() error {
	templates := map[string]string{
		"pkg/redis/client.go": "templates/monolith/pkg/redis/client.go.tmpl",
		"pkg/redis/cache.go":  "templates/monolith/pkg/redis/cache.go.tmpl",
	}

	return g.copyTemplates(templates)
}

// generateES 生成 Elasticsearch 组件
func (g *Generator) generateES() error {
	templates := map[string]string{
		"pkg/es/client.go": "templates/monolith/pkg/es/client.go.tmpl",
	}

	return g.copyTemplates(templates)
}

// generateMQ 生成消息队列组件
func (g *Generator) generateMQ() error {
	baseTemplates := map[string]string{
		"internal/mq/producer.go": "templates/monolith/internal/mq/producer.go.tmpl",
	}

	if err := g.copyTemplates(baseTemplates); err != nil {
		return err
	}

	// 根据 MQ 类型添加特定模板
	mqType := strings.ToLower(g.config.MQ)
	switch mqType {
	case "kafka":
		return g.copyTemplates(map[string]string{
			"internal/mq/kafka.go": "templates/monolith/internal/mq/kafka.go.tmpl",
		})
	case "rabbitmq":
		return g.copyTemplates(map[string]string{
			"internal/mq/rabbitmq.go": "templates/monolith/internal/mq/rabbitmq.go.tmpl",
		})
	case "rocketmq":
		return g.copyTemplates(map[string]string{
			"internal/mq/rocketmq.go": "templates/monolith/internal/mq/rocketmq.go.tmpl",
		})
	}

	return nil
}

// generateObjectStorage 生成对象存储组件
func (g *Generator) generateObjectStorage() error {
	templates := map[string]string{
		"internal/object/storage.go": "templates/monolith/internal/object/storage.go.tmpl",
	}

	// 根据配置添加特定的对象存储模板
	switch g.config.ObjectDB {
	case "oss":
		templates["internal/object/oss.go"] = "templates/monolith/internal/object/oss.go.tmpl"
	case "minio":
		templates["internal/object/minio.go"] = "templates/monolith/internal/object/minio.go.tmpl"
	case "rustfs":
		templates["internal/object/rustfs.go"] = "templates/monolith/internal/object/rustfs.go.tmpl"
	}

	return g.copyTemplates(templates)
}

// generateVector 生成向量数据库组件
func (g *Generator) generateVector() error {
	templates := map[string]string{
		"internal/vector/client.go": "templates/monolith/internal/vector/client.go.tmpl",
	}

	return g.copyTemplates(templates)
}

// copyTemplates 复制模板文件
func (g *Generator) copyTemplates(templates map[string]string) error {
	for dst, src := range templates {
		dstPath := filepath.Join(g.config.OutputDir, dst)
		srcPath := src

		// 检查源文件是否存在
		if _, err := os.Stat(srcPath); err != nil {
			continue // skip if template doesn't exist
		}

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		loader := template.NewLoader(filepath.Dir(srcPath))
		if err := loader.Load(""); err != nil {
			return err
		}

		processor := template.NewProcessor(loader)
		if err := processor.RenderFile(srcPath, dstPath, g.config); err != nil {
			return err
		}
	}

	return nil
}

// GenerateDocker 生成 Docker 相关文件
func (g *Generator) GenerateDocker(outputDir string) error {
	srcFiles := []string{
		"templates/monolith/docker-compose.yaml.tmpl",
		"templates/monolith/Dockerfile.tmpl",
	}

	for _, src := range srcFiles {
		_, err := os.Stat(src)
		if err != nil {
			continue // skip if template doesn't exist
		}
		dstName := filepath.Base(src[:len(src)-len(".tmpl")])
		dst := filepath.Join(outputDir, dstName)
		loader := template.NewLoader(filepath.Dir(src))
		if err := loader.Load(""); err != nil {
			return err
		}
		processor := template.NewProcessor(loader)
		if err := processor.RenderFile(src, dst, g.config); err != nil {
			return err
		}
	}

	return nil
}
