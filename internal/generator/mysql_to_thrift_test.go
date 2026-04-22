package generator

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMySQLToThriftGenerator_NewMySQLToThriftGenerator tests constructor
func TestMySQLToThriftGenerator_NewMySQLToThriftGenerator(t *testing.T) {
	dsn := "user:pass@tcp(localhost:3306)/db"
	tableName := "users"
	namespace := "example.v1"
	outputDir := "/tmp/test"

	gen := NewMySQLToThriftGenerator(dsn, tableName, namespace, outputDir)

	if gen.dsn != dsn {
		t.Errorf("expected dsn %s, got %s", dsn, gen.dsn)
	}
	if gen.tableName != tableName {
		t.Errorf("expected tableName %s, got %s", tableName, gen.tableName)
	}
	if gen.namespace != namespace {
		t.Errorf("expected namespace %s, got %s", namespace, gen.namespace)
	}
	if gen.outputDir != outputDir {
		t.Errorf("expected outputDir %s, got %s", outputDir, gen.outputDir)
	}
}

// TestMySQLToThriftType tests MySQL to Thrift type mapping
func TestMySQLToThriftType(t *testing.T) {
	gen := &MySQLToThriftGenerator{}

	tests := []struct {
		mysqlType   string
		nullable    bool
		expected    string
		expectError bool
	}{
		{"int", false, "i32", false},
		{"int", true, "optional i32", false},
		{"bigint", false, "i64", false},
		{"bigint", true, "optional i64", false},
		{"varchar(255)", false, "string", false},
		{"text", false, "string", false},
		{"double", false, "double", false},
		{"datetime", false, "i64", false},
		{"timestamp", true, "optional i64", false},
		{"json", false, "string", false},
		{"unknown", false, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.mysqlType, func(t *testing.T) {
			result, err := gen.mysqlToThriftType(tt.mysqlType, tt.nullable)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for type %s", tt.mysqlType)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("for MySQL type %s (nullable=%v), expected %s, got %s",
					tt.mysqlType, tt.nullable, tt.expected, result)
			}
		})
	}
}

// TestMySQLToThriftGenerator_RenderThriftIDL tests Thrift IDL rendering
func TestMySQLToThriftGenerator_RenderThriftIDL(t *testing.T) {
	tableInfo := &TableInfo{
		TableName:  "users",
		PrimaryKey: "id",
		Columns: []ColumnInfo{
			{Name: "id", Type: "bigint", Nullable: false, ThriftType: "i64", Comment: "Primary key"},
			{Name: "username", Type: "varchar(100)", Nullable: false, ThriftType: "string", Comment: "User name"},
			{Name: "email", Type: "varchar(255)", Nullable: true, ThriftType: "optional string", Comment: "Email address"},
			{Name: "created_at", Type: "timestamp", Nullable: false, ThriftType: "i64", Comment: "Created time"},
		},
	}

	gen := &MySQLToThriftGenerator{
		namespace: "example.v1",
	}

	output, err := gen.renderThriftIDL(tableInfo)
	if err != nil {
		t.Fatalf("renderThriftIDL failed: %v", err)
	}

	// Check for key elements
	checks := []string{
		"namespace go example.v1",
		"struct users",
		"struct GetusersReq",
		"struct CreateusersReq",
		"service usersService",
	}

	for _, check := range checks {
		if !containsString(output, check) {
			t.Errorf("rendered Thrift should contain: %s", check)
		}
	}
}

// TestMySQLToThriftGenerator_Generate tests end-to-end generation (requires mock DB)
func TestMySQLToThriftGenerator_Generate_Skip(t *testing.T) {
	t.Skip("Skipping integration test - requires MySQL database")

	gen := NewMySQLToThriftGenerator(
		"root:password@tcp(localhost:3306)/test_db",
		"test_table",
		"example.v1",
		"/tmp/test-output",
	)

	if err := gen.Generate(); err != nil {
		t.Logf("Generation failed (expected in CI): %v", err)
		return
	}

	thriftFile := filepath.Join("/tmp/test-output", "test_table.thrift")
	if _, err := os.Stat(thriftFile); err != nil {
		t.Errorf("Thrift file should exist: %v", err)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		containsStringHelper(s, substr, 0))
}

func containsStringHelper(s, substr string, start int) bool {
	if start > len(s)-len(substr) {
		return false
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsStringHelper(s, substr, start+1)
}
