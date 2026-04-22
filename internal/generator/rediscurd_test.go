package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedisCRUDGenerator_NewRedisCRUDGenerator tests constructor
func TestRedisCRUDGenerator_NewRedisCRUDGenerator(t *testing.T) {
	g := NewRedisCRUDGenerator("/tmp/test")
	assert.NotNil(t, g)
	assert.Equal(t, "/tmp/test", g.outputDir)
}

// TestRedisCRUDGenerator_toPascalCase tests toPascalCase helper
func TestRedisCRUDGenerator_toPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with t_ prefix", "t_user_order", "UserOrder"},
		{"without prefix", "product", "Product"},
		{"single word", "user", "User"},
		{"complex", "t_order_item_detail", "OrderItemDetail"},
		{"empty", "", ""},
		{"snake case", "order_item", "OrderItem"},
	}

	g := &RedisCRUDGenerator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.toPascalCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRedisCRUDGenerator_Generate tests Generate method
func TestRedisCRUDGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("user")
	assert.NoError(t, err)

	// Check directory created
	repoDir := filepath.Join(tmpDir, "internal/repository")
	_, err = os.Stat(repoDir)
	assert.NoError(t, err)

	// Check file created
	filePath := filepath.Join(repoDir, "user_redis_repository.go")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

// TestRedisCRUDGenerator_GenerateWithPrefix tests Generate with t_ prefix
func TestRedisCRUDGenerator_GenerateWithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("t_order_item")
	assert.NoError(t, err)

	// Check file created with correct name
	filePath := filepath.Join(tmpDir, "internal/repository", "orderitem_redis_repository.go")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Check file content
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "package repository")
	assert.Contains(t, string(content), "OrderItemRepository")
	assert.Contains(t, string(content), "NewOrderItemRepository")
}

// TestRedisCRUDGenerator_GenerateComplexName tests Generate with complex entity name
func TestRedisCRUDGenerator_GenerateComplexName(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("t_user_profile_data")
	assert.NoError(t, err)

	filePath := filepath.Join(tmpDir, "internal/repository", "userprofiledata_redis_repository.go")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

// TestRedisCRUDGenerator_GenerateRepositoryContent tests generated repository content
func TestRedisCRUDGenerator_GenerateRepositoryContent(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("product")
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "internal/repository", "product_redis_repository.go"))
	assert.NoError(t, err)
	contentStr := string(content)

	// Check imports
	assert.Contains(t, contentStr, "github.com/redis/go-redis/v9")
	assert.Contains(t, contentStr, "context")
	assert.Contains(t, contentStr, "encoding/json")

	// Check struct
	assert.Contains(t, contentStr, "type ProductRepository struct")
	assert.Contains(t, contentStr, "client *redis.Client")
	assert.Contains(t, contentStr, "prefix string")
	assert.Contains(t, contentStr, "ttl    time.Duration")

	// Check constructor
	assert.Contains(t, contentStr, "NewProductRepository")
	assert.Contains(t, contentStr, "client *redis.Client")
	assert.Contains(t, contentStr, "ttl time.Duration")

	// Check methods
	assert.Contains(t, contentStr, "Set")
	assert.Contains(t, contentStr, "Get")
	assert.Contains(t, contentStr, "Del")
	assert.Contains(t, contentStr, "Exists")
	assert.Contains(t, contentStr, "TTL")
	assert.Contains(t, contentStr, "Expire")
	assert.Contains(t, contentStr, "Keys")
	assert.Contains(t, contentStr, "GetKeys")
	assert.Contains(t, contentStr, "Count")
	assert.Contains(t, contentStr, "Clear")

	// Check key prefix
	assert.Contains(t, contentStr, "product:")
}

// TestRedisCRUDGenerator_GenerateMethodSignatures tests method signatures
func TestRedisCRUDGenerator_GenerateMethodSignatures(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("order")
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "internal/repository", "order_redis_repository.go"))
	assert.NoError(t, err)
	contentStr := string(content)

	// Check Set method
	assert.Contains(t, contentStr, "Set(ctx context.Context, id string, entity *Order) error")

	// Check Get method
	assert.Contains(t, contentStr, "Get(ctx context.Context, id string) (*Order, error)")

	// Check Del method
	assert.Contains(t, contentStr, "Del(ctx context.Context, id string) error")

	// Check Exists method
	assert.Contains(t, contentStr, "Exists(ctx context.Context, id string) (bool, error)")

	// Check TTL method
	assert.Contains(t, contentStr, "TTL(ctx context.Context, id string) (time.Duration, error)")

	// Check Expire method
	assert.Contains(t, contentStr, "Expire(ctx context.Context, id string, ttl time.Duration) error")

	// Check Keys method
	assert.Contains(t, contentStr, "Keys(ctx context.Context, pattern string) ([]string, error)")

	// Check Count method
	assert.Contains(t, contentStr, "Count(ctx context.Context) (int64, error)")
}

// TestRedisCRUDGenerator_GenerateErrorHandling tests error handling in generated code
func TestRedisCRUDGenerator_GenerateErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("user")
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "internal/repository", "user_redis_repository.go"))
	assert.NoError(t, err)
	contentStr := string(content)

	// Check error handling
	assert.Contains(t, contentStr, "redis.Nil")
	assert.Contains(t, contentStr, "not found")
	assert.Contains(t, contentStr, "if err != nil")
}

// TestRedisCRUDGenerator_GenerateJSONOperations tests JSON operations
func TestRedisCRUDGenerator_GenerateJSONOperations(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("user")
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "internal/repository", "user_redis_repository.go"))
	assert.NoError(t, err)
	contentStr := string(content)

	// Check JSON operations use encoding/json package
	assert.Contains(t, contentStr, "json.Marshal")
	assert.Contains(t, contentStr, "json.Unmarshal")
	assert.Contains(t, contentStr, "encoding/json")
}

// TestRedisCRUDGenerator_GenerateWithDifferentEntities tests multiple entity types
func TestRedisCRUDGenerator_GenerateWithDifferentEntities(t *testing.T) {
	entities := []string{"user", "product", "order", "t_category", "t_order_item"}

	for _, entity := range entities {
		t.Run(entity, func(t *testing.T) {
			tmpDir := t.TempDir()
			g := NewRedisCRUDGenerator(tmpDir)
			err := g.Generate(entity)
			assert.NoError(t, err)

			pascalCase := g.toPascalCase(entity)
			lowerCase := ToLowerCamelCase(pascalCase)
			filePath := filepath.Join(tmpDir, "internal/repository", lowerCase+"_redis_repository.go")
			_, err = os.Stat(filePath)
			assert.NoError(t, err)
		})
	}
}

// TestRedisCRUDGenerator_GenerateIntegration tests full integration
func TestRedisCRUDGenerator_GenerateIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	g := NewRedisCRUDGenerator(tmpDir)
	err := g.Generate("t_integration_test")
	assert.NoError(t, err)

	// Verify complete file structure
	expectedContent := []string{
		"package repository",
		"type IntegrationTestRepository struct",
		"func NewIntegrationTestRepository",
		"func (r *IntegrationTestRepository) Set",
		"func (r *IntegrationTestRepository) Get",
		"func (r *IntegrationTestRepository) Del",
		"integrationtest:",
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "internal/repository", "integrationtest_redis_repository.go"))
	assert.NoError(t, err)
	contentStr := string(content)

	for _, expected := range expectedContent {
		assert.Contains(t, contentStr, expected)
	}
}
