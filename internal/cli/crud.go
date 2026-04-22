package cli

import (
	"fmt"
	"strings"

	"github.com/gospacex/gpx/internal/generator"
	"github.com/spf13/cobra"
)

var (
	crudHost     string
	crudPort     string
	crudUser     string
	crudPassword string
	crudDB       string
	crudTable    string
	crudOutput   string
	crudModule   string
)

var crudCmd = &cobra.Command{
	Use:   "crud",
	Short: "Generate CRUD code from MySQL table structure",
	Long: `Connect to MySQL, read table structure, automatically generate three-layer CRUD code:
model → repository → service → handler (gin)

Examples:

  # Generate CRUD from existing table
  gpx crud \
    --host 127.0.0.1 \
    --port 3306 \
    --user root \
    --password secret \
    --database mydb \
    --table users \
    --output internal \
    --module github.com/yourorg/yourproject`,
	RunE: runCRUD,
}

func runCRUD(cmd *cobra.Command, args []string) error {
	if crudDB == "" || crudTable == "" {
		return fmt.Errorf("database and table are required")
	}
	if crudHost == "" {
		crudHost = "127.0.0.1"
	}
	if crudPort == "" {
		crudPort = "3306"
	}
	if crudOutput == "" {
		crudOutput = "./internal"
	}

	gen := generator.NewCRUDGenerator(
		crudHost,
		crudPort,
		crudUser,
		crudPassword,
		crudDB,
		crudTable,
		crudOutput,
		crudModule,
	)

	if err := gen.Generate(); err != nil {
		return err
	}

	fmt.Printf("✓ CRUD code generated successfully for table %s.%s\n", crudDB, crudTable)
	fmt.Printf("  Output directory: %s/%s\n", crudOutput, crudTable)
	fmt.Println()
	fmt.Println("Generated files:")
	fmt.Printf("  - %s/%s/model/%s_model.go\n", crudOutput, crudTable, camelCase(crudTable))
	fmt.Printf("  - %s/%s/repository/%s_repository.go\n", crudOutput, crudTable, camelCase(crudTable))
	fmt.Printf("  - %s/%s/service/%s_service.go\n", crudOutput, crudTable, camelCase(crudTable))
	fmt.Printf("  - %s/%s/handler/%s_handler.go\n", crudOutput, crudTable, camelCase(crudTable))

	return nil
}

func camelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	// keep the code simple, first char lowercase
	words := strings.Split(s, "_")
	for i, w := range words {
		if i == 0 {
			words[i] = strings.ToLower(w)
		} else {
			words[i] = strings.Title(strings.ToLower(w))
		}
	}
	return strings.Join(words, "")
}

func init() {
	// Flags
	crudCmd.Flags().StringVar(&crudHost, "host", "", "MySQL host")
	crudCmd.Flags().StringVar(&crudPort, "port", "", "MySQL port")
	crudCmd.Flags().StringVar(&crudUser, "user", "", "MySQL username")
	crudCmd.Flags().StringVar(&crudPassword, "password", "", "MySQL password")
	crudCmd.Flags().StringVar(&crudDB, "database", "", "MySQL database name (required)")
	crudCmd.Flags().StringVar(&crudTable, "table", "", "MySQL table name (required)")
	crudCmd.Flags().StringVar(&crudOutput, "output", "", "Output directory (default: internal)")
	crudCmd.Flags().StringVar(&crudModule, "module", "", "Go module name for import (required for import path)")

	// Mark required flags
	_ = crudCmd.MarkFlagRequired("database")
	_ = crudCmd.MarkFlagRequired("table")
	_ = crudCmd.MarkFlagRequired("module")
}

// GetCRUDCmd returns the crud command
func GetCRUDCmd() *cobra.Command {
	return crudCmd
}
