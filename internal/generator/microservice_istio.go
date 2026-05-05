package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// MicroserviceIstioGenerator Istio 微服务生成器
type MicroserviceIstioGenerator struct {
	serviceName string
	outputDir   string
}

// NewMicroserviceIstioGenerator creates new Istio microservice generator
func NewMicroserviceIstioGenerator(serviceName, outputDir string) *MicroserviceIstioGenerator {
	return &MicroserviceIstioGenerator{
		serviceName: serviceName,
		outputDir:   outputDir,
	}
}

// Generate generates Istio microservice project
func (g *MicroserviceIstioGenerator) Generate() error {
	dirs := []string{
		"app",
		"manifest/bookinfo",
		"manifest/istio",
		"deploy/k8s",
		"configs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"app/main.go":                  g.mainContent(),
		"app/handler.go":               g.handlerContent(),
		"manifest/bookinfo/service.yaml": g.serviceContent(),
		"manifest/bookinfo/deployment.yaml": g.deploymentContent(),
		"manifest/istio/virtual-service.yaml": g.virtualServiceContent(),
		"manifest/istio/destination-rule.yaml": g.destinationRuleContent(),
		"manifest/istio/gateway.yaml":  g.gatewayContent(),
		"deploy/k8s/kustomization.yaml": g.kustomizeContent(),
		"go.mod":                       g.goModContent(),
		"readme.md":                    g.readmeContent(),
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

func (g *MicroserviceIstioGenerator) mainContent() string {
	return fmt.Sprintf(`package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/", apiHandler)
	
	log.Printf("Starting %s on port %%s", port)
	http.ListenAndServe(":"+port, nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(` + "`" + `{"service":"%s"}` + "`" + `))
}
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) handlerContent() string {
	return `package main

import (
	"context"
	"log"
)

// Handler business handler
type Handler struct{}

// NewHandler creates handler
func NewHandler() *Handler {
	return &Handler{}
}

// Process processes request
func (h *Handler) Process(ctx context.Context, input string) (string, error) {
	log.Printf("Processing: %%s", input)
	return "Processed: " + input, nil
}
`
}

func (g *MicroserviceIstioGenerator) serviceContent() string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  labels:
    app: %s
    version: v1
spec:
  ports:
  - port: 8080
    name: http
  selector:
    app: %s
`, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) deploymentContent() string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  labels:
    app: %s
spec:
  replicas: 3
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
        version: v1
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) virtualServiceContent() string {
	return fmt.Sprintf(`apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: %s
spec:
  hosts:
  - %s
  http:
  - match:
    - headers:
        version:
          exact: v2
    route:
    - destination:
        host: %s
        subset: v2
    weight: 100
  - route:
    - destination:
        host: %s
        subset: v1
      weight: 90
    - destination:
        host: %s
        subset: v2
      weight: 10
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) destinationRuleContent() string {
	return fmt.Sprintf(`apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: %s
spec:
  host: %s
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        h2UpgradePolicy: UPGRADE
        http1MaxPendingRequests: 100
        http2MaxRequests: 1000
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
`, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) gatewayContent() string {
	return fmt.Sprintf(`apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: %s-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: %s-gateway-vs
spec:
  hosts:
  - "*"
  gateways:
  - %s-gateway
  http:
  - match:
    - uri:
        prefix: /api
    route:
    - destination:
        host: %s
        port:
          number: 8080
`, g.serviceName, g.serviceName, g.serviceName, g.serviceName)
}

func (g *MicroserviceIstioGenerator) kustomizeContent() string {
	return `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../bookinfo
  - ../istio

commonLabels:
  managed-by: gpx
`
}

func (g *MicroserviceIstioGenerator) goModContent() string {
	return fmt.Sprintf(`module %s

go %s
`, g.serviceName, GetGoVersion())
}

func (g *MicroserviceIstioGenerator) readmeContent() string {
	return fmt.Sprintf(`# %s - Istio Service Mesh

基于 Istio 的服务网格项目

## 部署

### 基础服务

kubectl apply -f manifest/bookinfo/

### Istio 配置

kubectl apply -f manifest/istio/

### 使用 Kustomize

kubectl apply -k deploy/k8s/

## Istio 配置

- **VirtualService**: 流量路由（v1/v2 灰度）
- **DestinationRule**: 流量策略（连接池/子集）
- **Gateway**: 入口网关

## 灰度发布

v1: 90%% 流量
v2: 10%% 流量 (通过 header version=v2 访问 100%%)

## 参考

bookinfo Istio 官方示例
`, g.serviceName)
}
