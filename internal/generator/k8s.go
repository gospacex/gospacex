package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// K8sGenerator Kubernetes 资源生成器
type K8sGenerator struct{}

// NewK8sGenerator creates new K8s generator
func NewK8sGenerator() *K8sGenerator {
	return &K8sGenerator{}
}

// Generate 生成 k8s 资源
func (k *K8sGenerator) Generate(outputDir string, projectName string) error {
	// 创建目录
	if err := os.MkdirAll(filepath.Join(outputDir, "deploy/k8s"), 0o755); err != nil {
		return err
	}

	// 生成 CronJob
	if err := k.generateCronJob(outputDir, projectName); err != nil {
		return err
	}

	// 生成 ConfigMap
	if err := k.generateConfigMap(outputDir, projectName); err != nil {
		return err
	}

	// 生成 Deployment（可选，用于常驻服务）
	if err := k.generateDeployment(outputDir, projectName); err != nil {
		return err
	}

	// 生成 kustomize 配置
	if err := k.generateKustomize(outputDir); err != nil {
		return err
	}

	return nil
}

// generateCronJob 生成 CronJob YAML
func (k *K8sGenerator) generateCronJob(outputDir, projectName string) error {
	content := fmt.Sprintf(`# CronJob for task scheduling
# Usage: kubectl apply -f cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: %s-scheduler
  namespace: default
  labels:
    app: %s
    component: scheduler
spec:
  # Cron 表达式（每分钟执行一次）
  schedule: "*/1 * * * *"
  
  # 时区（需要 k8s 1.27+）
  timeZone: "Asia/Shanghai"
  
  # 并发策略
  concurrencyPolicy: Forbid
  
  # 失败重试次数
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  
  # Job 模板
  jobTemplate:
    spec:
      # 超时时间
      activeDeadlineSeconds: 300
      
      # 重试次数
      backoffLimit: 2
      
      template:
        spec:
          restartPolicy: OnFailure
          
          containers:
          - name: task-runner
            image: %s:latest
            imagePullPolicy: IfNotPresent
            
            # 运行任务的命令
            command: ["/app/%s"]
            args: ["run", "example_task"]
            
            # 环境变量
            env:
            - name: CONFIG_PATH
              value: "/etc/%s/config.yaml"
            - name: LOG_LEVEL
              value: "info"
            
            # 资源限制
            resources:
              requests:
                cpu: "100m"
                memory: "128Mi"
              limits:
                cpu: "500m"
                memory: "512Mi"
            
            # 挂载配置
            volumeMounts:
            - name: config
              mountPath: /etc/%s
              readOnly: true
            - name: logs
              mountPath: /var/log/%s
          
          volumes:
          - name: config
            configMap:
              name: %s-config
          - name: logs
            emptyDir: {}
`,
		projectName, projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
	)

	return os.WriteFile(
		filepath.Join(outputDir, "deploy/k8s/cronjob.yaml"),
		[]byte(content),
		0o644,
	)
}

// generateConfigMap 生成 ConfigMap
func (k *K8sGenerator) generateConfigMap(outputDir, projectName string) error {
	content := fmt.Sprintf(`# ConfigMap for application configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: default
  labels:
    app: %s
data:
  config.yaml: |
    # Application configuration
    app:
      name: %s
      environment: production
    
    # Database configuration
    database:
      mysql:
        dsn: "${MYSQL_DSN}"
      redis:
        addr: "${REDIS_ADDR}"
    
    # Logging
    log:
      level: info
      format: json
    
    # Scheduler (using ouqiang/gocron platform)
    # Note: Tasks are managed via gocron web UI
    # This config is for task runner only
    scheduler:
      type: external
      gocron_url: "http://gocron:8080"
`,
		projectName, projectName, projectName,
	)

	return os.WriteFile(
		filepath.Join(outputDir, "deploy/k8s/configmap.yaml"),
		[]byte(content),
		0o644,
	)
}

// generateDeployment 生成 Deployment（用于常驻服务）
func (k *K8sGenerator) generateDeployment(outputDir, projectName string) error {
	content := fmt.Sprintf(`# Deployment for long-running services
# Use this if you need a常驻 service (not just scheduled tasks)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: default
  labels:
    app: %s
spec:
  replicas: 2
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        imagePullPolicy: IfNotPresent
        
        ports:
        - containerPort: 8080
          name: http
        
        # Health checks
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
        
        env:
        - name: CONFIG_PATH
          value: "/etc/%s/config.yaml"
        
        volumeMounts:
        - name: config
          mountPath: /etc/%s
          readOnly: true
      
      volumes:
      - name: config
        configMap:
          name: %s-config

---
# Service for accessing the deployment
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: default
  labels:
    app: %s
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
    name: http
  selector:
    app: %s
`,
		projectName, projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
		projectName,
	)

	return os.WriteFile(
		filepath.Join(outputDir, "deploy/k8s/deployment.yaml"),
		[]byte(content),
		0o644,
	)
}

// generateKustomize 生成 kustomize 配置
func (k *K8sGenerator) generateKustomize(outputDir string) error {
	content := `# Kustomize configuration
# Usage: kubectl apply -k deploy/k8s/
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - cronjob.yaml
  - configmap.yaml
  - deployment.yaml

# Common labels
commonLabels:
  managed-by: gpx

# Namespace
namespace: default

# ConfigMap generator
configMapGenerator:
  - name: app-version
    literals:
      - VERSION=0.1.0
`

	return os.WriteFile(
		filepath.Join(outputDir, "deploy/k8s/kustomization.yaml"),
		[]byte(content),
		0o644,
	)
}

// GenerateSchedulerDoc 生成调度器说明文档
func (k *K8sGenerator) GenerateSchedulerDoc(outputDir string) error {
	content := `# 任务调度说明

## 调度模式

本项目支持两种调度模式：

## 1. gocron 平台调度（推荐）

使用 https://github.com/ouqiang/gocron 进行任务调度。

部署 gocron:

	docker run -d --name gocron -p 8080:8080 -v /data/gocron:/data ouqiang/gocron

配置任务:
1. 访问 http://localhost:8080
2. 创建任务，配置 Cron 表达式
3. 设置执行命令

优势:
- Web UI 管理任务
- 任务执行日志
- 失败告警
- 任务依赖管理

## 2. k8s CronJob 调度

使用 Kubernetes CronJob 进行调度。

部署:
	kubectl apply -k deploy/k8s/

查看 CronJob:
	kubectl get cronjobs
	kubectl get jobs
	kubectl get pods

手动触发任务:
	kubectl create job --from=cronjob/cronjob-name manual-run

## 选择建议

| 场景 | 推荐方案 |
|------|---------|
| 开发环境 | gocron 平台 |
| 小规模生产 | gocron 平台 |
| k8s 集群 | CronJob |
| 需要高可用 | CronJob + 多副本 |
| 复杂任务依赖 | gocron 平台 |
`

	return os.WriteFile(
		filepath.Join(outputDir, "deploy/SCHEDULER.md"),
		[]byte(content),
		0o644,
	)
}
