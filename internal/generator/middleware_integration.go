package generator

import (
	
	"os"
	"path/filepath"
)

// MiddlewareIntegrationGenerator 中间件集成生成器
type MiddlewareIntegrationGenerator struct {
	outputDir   string
	middlewares []string
}

// NewMiddlewareIntegrationGenerator creates new middleware integration generator
func NewMiddlewareIntegrationGenerator(outputDir string, middlewares []string) *MiddlewareIntegrationGenerator {
	return &MiddlewareIntegrationGenerator{
		outputDir:   outputDir,
		middlewares: middlewares,
	}
}

// Generate generates middleware integration code
func (g *MiddlewareIntegrationGenerator) Generate() error {
	dirs := []string{
		"internal/middleware/jaeger",
		"internal/middleware/kafka",
		"internal/middleware/rocketmq",
		"internal/middleware/nacos",
		"internal/middleware/apollo",
		"internal/middleware/consul",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.outputDir, dir), 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		"internal/middleware/jaeger/init.go":     g.jaegerContent(),
		"internal/middleware/kafka/init.go":      g.kafkaContent(),
		"internal/middleware/rocketmq/init.go":   g.rocketmqContent(),
		"internal/middleware/nacos/init.go":      g.nacosContent(),
		"internal/middleware/apollo/init.go":     g.apolloContent(),
		"internal/middleware/consul/init.go":     g.consulContent(),
		"internal/middleware/registry.go":        g.registryContent(),
		"configs/middleware.yaml":                g.middlewareConfigContent(),
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

func (g *MiddlewareIntegrationGenerator) jaegerContent() string {
	return `package jaeger

import (
	"io"
	"os"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

var closer io.Closer

// Init initializes Jaeger tracing
func Init(serviceName string) {
	cfg := &config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: getEnv("JAEGER_ADDR", "localhost:6831"),
		},
	}
	
	tracer, c, err := cfg.NewTracer()
	if err != nil { panic(err) }
	
	closer = c
	// Set global tracer
	// opentracing.SetGlobalTracer(tracer)
}

// Close closes tracing
func Close() {
	if closer != nil { closer.Close() }
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *MiddlewareIntegrationGenerator) kafkaContent() string {
	return `package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	"os"
	"strings"
)

var Producer *kafka.Writer
var Consumer *kafka.Reader

// Init initializes Kafka
func Init() {
	brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	
	Producer = &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}
	
	Consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   "my-topic",
	})
}

// Close closes connections
func Close() {
	if Producer != nil { Producer.Close() }
	if Consumer != nil { Consumer.Close() }
}

// Produce sends message
func Produce(ctx context.Context, topic, key, value string) error {
	return Producer.WriteMessages(ctx,
		kafka.Message{
			Topic: topic,
			Key:   []byte(key),
			Value: []byte(value),
		},
	)
}

// Consume consumes message
func Consume(ctx context.Context) (kafka.Message, error) {
	return Consumer.FetchMessage(ctx)
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *MiddlewareIntegrationGenerator) rocketmqContent() string {
	return `package rocketmq

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

var Producer rocketmq.Producer
var Consumer rocketmq.PushConsumer

// Init initializes RocketMQ
func Init() {
	// Initialize Producer
	var err error
	Producer, err = rocketmq.NewProducer(
		&primitive.ProducerConfig{
			Resolver:      primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"}),
			ProducerGroup: "my-producer-group",
		},
	)
	if err != nil { panic(err) }
	Producer.Start()
	
	// Initialize Consumer
	Consumer, err = rocketmq.NewPushConsumer(
		consumer.WithGroupName("my-consumer-group"),
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"})),
	)
	if err != nil { panic(err) }
}

// Close closes connections
func Close() {
	if Producer != nil { Producer.Shutdown() }
	if Consumer != nil { Consumer.Shutdown() }
}

// Produce sends message
func Produce(ctx context.Context, topic string, body []byte) error {
	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}
	_, err := Producer.SendSync(ctx, msg)
	return err
}

// Subscribe subscribes topic
func Subscribe(topic string, f func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)) error {
	return Consumer.Subscribe(topic, f)
}

// StartConsumer starts consumer
func StartConsumer() error {
	return Consumer.Start()
}
`
}

func (g *MiddlewareIntegrationGenerator) nacosContent() string {
	return `package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

var ConfigClient config_client.IConfigClient

// Init initializes Nacos
func Init() {
	sc := []constant.ServerConfig{
		*constant.NewServerConfig("127.0.0.1", 8848),
	}
	
	cc := &constant.ClientConfig{
		NamespaceId:         "public",
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "debug",
	}
	
	var err error
	ConfigClient, err = clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
	if err != nil { panic(err) }
}

// GetConfig gets config
func GetConfig(dataID, group string) (string, error) {
	return ConfigClient.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
	})
}

// PublishConfig publishes config
func PublishConfig(dataID, group, content string) error {
	return ConfigClient.PublishConfig(vo.ConfigParam{
		DataId:  dataID,
		Group:   group,
		Content: content,
	})
}

// ListenConfig listens config changes
func ListenConfig(dataID, group string, onChange func(namespace, group, dataId, data string)) error {
	return ConfigClient.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
		OnChange: onChange,
	})
}
`
}

func (g *MiddlewareIntegrationGenerator) apolloContent() string {
	return `package apollo

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
)

var Client agollo.Client

// Init initializes Apollo
func Init() {
	c := &config.AppConfig{
		AppID:          "my-app",
		Cluster:        "default",
		NamespaceName:  "application",
		IP:             "http://localhost:8080",
		IsBackupConfig: true,
	}
	
	var err error
	Client, err = agollo.StartWithConfig(func() (*config.AppConfig, error) { return c, nil })
	if err != nil { panic(err) }
}

// GetConfig gets config value
func GetConfig(key string) string {
	return Client.GetConfig("application").GetValue(key)
}

// GetConfigWithDefault gets config with default value
func GetConfigWithDefault(key, defaultValue string) string {
	v := Client.GetConfig("application").GetValue(key)
	if v == "" { return defaultValue }
	return v
}

// Listen listens config changes
func Listen(namespace string, onChange func(key, value string)) {
	Client.ListenToChannel(func(event *agollo.ChangeEvent) {
		for k, v := range event.Changes {
			onChange(k, v.NewValue)
		}
	})
}
`
}

func (g *MiddlewareIntegrationGenerator) consulContent() string {
	return `package consul

import (
	"context"
	"github.com/hashicorp/consul/api"
	"os"
)

var Client *api.Client

// Init initializes Consul
func Init() {
	cfg := api.DefaultConfig()
	cfg.Address = getEnv("CONSUL_ADDR", "localhost:8500")
	
	var err error
	Client, err = api.NewClient(cfg)
	if err != nil { panic(err) }
}

// Register registers service
func Register(name, id, address string, port int) error {
	return Client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Address: address,
		Port:    port,
		Check: &api.AgentServiceCheck{
			HTTP:     "http://" + address + ":" + string(rune(port)) + "/health",
			Interval: "10s",
			Timeout:  "5s",
		},
	})
}

// Deregister deregisters service
func Deregister(id string) error {
	return Client.Agent().ServiceDeregister(id)
}

// Discover discovers services
func Discover(name string) ([]*api.ServiceEntry, error) {
	entries, _, err := Client.Health().Service(name, "", true, nil)
	return entries, err
}

// Get gets key-value
func Get(key string) (string, error) {
	kv, _, err := Client.KV().Get(key, nil)
	if err != nil { return "", err }
	if kv == nil { return "", nil }
	return string(kv.Value), nil
}

// Put puts key-value
func Put(key, value string) error {
	_, err := Client.KV().Put(&api.KVPair{Key: key, Value: []byte(value)}, nil)
	return err
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}
`
}

func (g *MiddlewareIntegrationGenerator) registryContent() string {
	return `package middleware

import (
	"%s/internal/middleware/jaeger"
	"%s/internal/middleware/kafka"
	"%s/internal/middleware/rocketmq"
	"%s/internal/middleware/nacos"
	"%s/internal/middleware/apollo"
	"%s/internal/middleware/consul"
)

// Init initializes all middleware
func Init(serviceName string) {
	jaeger.Init(serviceName)
	kafka.Init()
	rocketmq.Init()
	nacos.Init()
	apollo.Init()
	consul.Init()
}

// Close closes all middleware
func Close() {
	jaeger.Close()
	kafka.Close()
	rocketmq.Close()
}
`
}

func (g *MiddlewareIntegrationGenerator) middlewareConfigContent() string {
	return `# Middleware Configuration

jaeger:
  addr: localhost:6831
  service_name: my-service

kafka:
  brokers: localhost:9092
  topic: my-topic

rocketmq:
  namesrv: 127.0.0.1:9876
  producer_group: my-producer-group
  consumer_group: my-consumer-group

nacos:
  addr: 127.0.0.1:8848
  namespace: public

apollo:
  addr: http://localhost:8080
  app_id: my-app
  cluster: default
  namespace: application

consul:
  addr: localhost:8500
`
}
