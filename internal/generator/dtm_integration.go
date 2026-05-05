package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// DTMIntegrationGenerator DTM 集成生成器
type DTMIntegrationGenerator struct {
	serviceName string
	outputDir   string
}

// NewDTMIntegrationGenerator creates new DTM integration generator
func NewDTMIntegrationGenerator(serviceName, outputDir string) *DTMIntegrationGenerator {
	return &DTMIntegrationGenerator{
		serviceName: serviceName,
		outputDir:   outputDir,
	}
}

// Generate generates DTM integration code
func (g *DTMIntegrationGenerator) Generate() error {
	dirs := []string{
		"internal/dtm",
		"internal/dtm/saga",
		"internal/dtm/tcc",
		"internal/dtm/workflow",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"internal/dtm/client.go":          g.clientContent(),
		"internal/dtm/saga/order.go":      g.sagaOrderContent(),
		"internal/dtm/tcc/order.go":       g.tccOrderContent(),
		"internal/dtm/workflow/order.go":  g.workflowOrderContent(),
		"configs/dtm.yaml":                g.dtmConfigContent(),
		"readme_dtm.md":                   g.readmeContent(),
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

func (g *DTMIntegrationGenerator) clientContent() string {
	return `package dtm

import (
	"os"
	"github.com/dtm-labs/dtm/client/dtmcli"
)

var DTMClient *dtmcli.DtmClient

// Init initializes DTM client
func Init() {
	dtmServer := getEnv("DTM_SERVER", "http://localhost:36789")
	DTMClient = dtmcli.NewDtmClient(dtmServer)
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *DTMIntegrationGenerator) sagaOrderContent() string {
	return fmt.Sprintf(`package saga

import (
	"context"
	"github.com/dtm-labs/dtm/client/dtmcli"
	"%s/internal/dtm"
)

// CreateOrderSaga creates order using SAGA pattern
func CreateOrderSaga(ctx context.Context, orderID string) error {
	saga := dtm.DTMClient.NewSaga("CreateOrder")
	
	saga.Add(dtmcli.GenActionURL("http://order-service/api/create"),
		dtmcli.GenActionURL("http://order-service/api/cancel"),
		map[string]interface{}{"order_id": orderID})
	
	saga.Add(dtmcli.GenActionURL("http://inventory-service/api/deduct"),
		dtmcli.GenActionURL("http://inventory-service/api/restore"),
		map[string]interface{}{"order_id": orderID})
	
	saga.Add(dtmcli.GenActionURL("http://payment-service/api/deduct"),
		dtmcli.GenActionURL("http://payment-service/api/restore"),
		map[string]interface{}{"order_id": orderID})
	
	return saga.Submit()
}
`, g.serviceName)
}

func (g *DTMIntegrationGenerator) tccOrderContent() string {
	return fmt.Sprintf(`package tcc

import (
	"context"
	"github.com/dtm-labs/dtm/client/dtmcli"
	"%s/internal/dtm"
)

// CreateOrderTCC creates order using TCC pattern
func CreateOrderTCC(ctx context.Context, orderID string) error {
	tcc := dtm.DTMClient.NewTCC("CreateOrderTCC")
	
	err := tcc.CallBranch(
		map[string]interface{}{"order_id": orderID},
		"http://order-service/api/try",
		"http://order-service/api/confirm",
		"http://order-service/api/cancel",
	)
	
	return err
}
`, g.serviceName)
}

func (g *DTMIntegrationGenerator) workflowOrderContent() string {
	return fmt.Sprintf(`package workflow

import (
	"context"
	"github.com/dtm-labs/dtm/client/workflow"
	"%s/internal/dtm"
)

// CreateOrderWorkflow creates order using workflow pattern
func CreateOrderWorkflow(ctx context.Context, orderID string) error {
	wf := dtm.DTMClient.NewWorkflow("CreateOrderWorkflow")
	
	return wf.Execute(func(w *workflow.Workflow) error {
		_, err := w.CallBranch(context.Background(),
			map[string]interface{}{"order_id": orderID},
			"http://order-service/api/create")
		if err != nil { return err }
		
		_, err = w.CallBranch(context.Background(),
			map[string]interface{}{"order_id": orderID},
			"http://inventory-service/api/deduct")
		if err != nil { return err }
		
		_, err = w.CallBranch(context.Background(),
			map[string]interface{}{"order_id": orderID},
			"http://payment-service/api/deduct")
		return err
	})
}
`, g.serviceName)
}

func (g *DTMIntegrationGenerator) dtmConfigContent() string {
	return `# DTM Configuration

dtm:
  server: http://localhost:36789
  
saga:
  enabled: true
  retry_limit: 3
  timeout: 60s

tcc:
  enabled: true
  timeout: 30s

workflow:
  enabled: true
  retry_limit: 3
`
}

func (g *DTMIntegrationGenerator) readmeContent() string {
	return `# DTM 分布式事务集成

## 支持的模式

1. **SAGA** - 长事务模式
2. **TCC** - Try-Confirm-Cancel
3. **Workflow** - 工作流模式

## 使用示例

### SAGA
` + "```go" + `
saga.CreateOrderSaga(ctx, "order-123")
` + "```" + `

### TCC
` + "```go" + `
tcc.CreateOrderTCC(ctx, "order-123")
` + "```" + `

### Workflow
` + "```go" + `
workflow.CreateOrderWorkflow(ctx, "order-123")
` + "```" + `

## 部署 DTM

docker run -d --name dtm -p 36789:36789 yedf/dtm
`
}
