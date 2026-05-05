package generator

import (
	"os"
	"path/filepath"
	"fmt"
)

type ConfigSystemGenerator struct {
	serviceName string
	outputDir   string
	remoteType  string
}

func NewConfigSystemGenerator(serviceName, outputDir, remoteType string) *ConfigSystemGenerator {
	if remoteType == "" { remoteType = "nacos" }
	return &ConfigSystemGenerator{serviceName: serviceName, outputDir: outputDir, remoteType: remoteType}
}

func (c *ConfigSystemGenerator) Build() error {
	dirs := []string{"pkg/config", "pkg/config/remote", "pkg/config/local", "configs"}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(c.outputDir, dir), 0o755)
	}
	
	files := map[string]string{
		"pkg/config/config.go": c.configCore(),
		"pkg/config/priority.go": c.priority(),
		"pkg/config/options.go": c.options(),
		"pkg/config/remote/nacos.go": c.nacosLoader(),
		"pkg/config/remote/apollo.go": c.apolloLoader(),
		"pkg/config/remote/consul.go": c.consulLoader(),
		"pkg/config/local/viper.go": c.viperLoader(),
		"pkg/config/local/env.go": c.envLoader(),
		"configs/config.yaml": c.configYAML(),
		"configs/config.local.yaml": c.localYAML(),
		".env.example": c.envExample(),
	}
	
	for path, content := range files {
		fullPath := filepath.Join(c.outputDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}
	return nil
}

func (c *ConfigSystemGenerator) configCore() string {
	return fmt.Sprintf(`package config

import (
	"sync"
	"os"
	"fmt"
	"%s/pkg/config/local"
)

type Config struct {
	mu sync.RWMutex
	loaders []Loader
	watchers map[string][]func(string, interface{})
	remote Loader
}

type Loader interface {
	Name() string
	Priority() Priority
	Load(key string) (interface{}, bool)
	Watch(key string, cb func(string, interface{})) error
	Close() error
}

var global *Config

func Init(opts ...Option) {
	cfg := &Options{}
	for _, opt := range opts { opt(cfg) }
	InitWithOptions(cfg)
}

func InitWithOptions(opts *Options) {
	global = &Config{
		watchers: make(map[string][]func(string, interface{})),
		loaders: make([]Loader, 0, 3),
	}
	
	if opts.RemoteType != "" {
		switch opts.RemoteType {
		case "nacos":
			global.remote = NewNacosLoader(opts.NacosAddr, opts.NacosDataID, opts.NacosGroup, opts.NacosNS)
		case "apollo":
			global.remote = NewApolloLoader(opts.ApolloAddr, opts.ApolloAppID, opts.ApolloNS)
		case "consul":
			global.remote = NewConsulLoader(opts.ConsulAddr, opts.ConsulKey)
		default:
			global.remote = NewNacosLoader(opts.NacosAddr, opts.NacosDataID, opts.NacosGroup, opts.NacosNS)
		}
		global.RegisterLoader(global.remote)
	}
	
	global.RegisterLoader(local.NewEnvLoader())
	global.RegisterLoader(local.NewViperLoader(opts.ConfigPath))
}

func Get(key string) interface{} { return global.Get(key) }
func GetString(key string) string { return global.GetString(key) }
func GetInt(key string) int { return global.GetInt(key) }
func GetFloat(key string) float64 { return global.GetFloat(key) }
func GetBool(key string) bool { return global.GetBool(key) }
func GetStringSlice(key string) []string { return global.GetStringSlice(key) }
func GetMap(key string) map[string]interface{} { return global.GetMap(key) }
func Watch(key string, cb func(string, interface{})) { global.Watch(key, cb) }
func Close() { global.Close() }

func (c *Config) RegisterLoader(loader Loader) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaders = append(c.loaders, loader)
}

func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, loader := range c.loaders {
		if v, ok := loader.Load(key); ok { return v }
	}
	return nil
}

func (c *Config) GetString(key string) string {
	if v := c.Get(key); v != nil {
		if s, ok := v.(string); ok { return s }
	}
	return ""
}

func (c *Config) GetInt(key string) int {
	if v := c.Get(key); v != nil {
		switch val := v.(type) {
		case int: return val
		case int64: return int(val)
		case float64: return int(val)
		case string:
			var i int
			fmt.Sscanf(val, "%%d", &i)
			return i
		}
	}
	return 0
}

func (c *Config) GetFloat(key string) float64 {
	if v := c.Get(key); v != nil {
		switch val := v.(type) {
		case float64: return val
		case int: return float64(val)
		case string:
			var f float64
			fmt.Sscanf(val, "%%f", &f)
			return f
		}
	}
	return 0
}

func (c *Config) GetBool(key string) bool {
	if v := c.Get(key); v != nil {
		if b, ok := v.(bool); ok { return b }
		if s, ok := v.(string); ok { return s == "true" || s == "1" || s == "yes" }
	}
	return false
}

func (c *Config) GetStringSlice(key string) []string {
	if v := c.Get(key); v != nil {
		if slice, ok := v.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, item := range slice {
				if s, ok := item.(string); ok { result[i] = s }
			}
			return result
		}
	}
	return []string{}
}

func (c *Config) GetMap(key string) map[string]interface{} {
	if v := c.Get(key); v != nil {
		if m, ok := v.(map[string]interface{}); ok { return m }
	}
	return make(map[string]interface{})
}

func (c *Config) Watch(key string, cb func(string, interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers[key] = append(c.watchers[key], cb)
	if c.remote != nil { c.remote.Watch(key, cb) }
}

func (c *Config) notifyWatchers(key string, value interface{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, cb := range c.watchers[key] { go cb(key, value) }
}

func (c *Config) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, loader := range c.loaders { loader.Close() }
}
`, c.serviceName)
}

func (c *ConfigSystemGenerator) priority() string {
	return `package config

type Priority int

const (
	PriorityRemote Priority = iota + 1
	PriorityEnv
	PriorityFile
	PriorityDefault
)

func (p Priority) String() string {
	switch p {
	case PriorityRemote: return "remote"
	case PriorityEnv: return "env"
	case PriorityFile: return "file"
	case PriorityDefault: return "default"
	default: return "unknown"
	}
}
`
}

func (c *ConfigSystemGenerator) options() string {
	return `package config

type Options struct {
	RemoteType   string
	NacosAddr    string
	NacosDataID  string
	NacosGroup   string
	NacosNS      string
	ApolloAddr   string
	ApolloAppID  string
	ApolloNS     string
	ConsulAddr   string
	ConsulKey    string
	ConfigPath   string
}

type Option func(*Options)

func WithRemoteType(typ string) Option {
	return func(o *Options) { o.RemoteType = typ }
}

func WithNacos(addr, dataID, group, ns string) Option {
	return func(o *Options) {
		o.RemoteType = "nacos"
		o.NacosAddr = addr
		o.NacosDataID = dataID
		o.NacosGroup = group
		o.NacosNS = ns
	}
}

func WithApollo(addr, appID, ns string) Option {
	return func(o *Options) {
		o.RemoteType = "apollo"
		o.ApolloAddr = addr
		o.ApolloAppID = appID
		o.ApolloNS = ns
	}
}

func WithConsul(addr, key string) Option {
	return func(o *Options) {
		o.RemoteType = "consul"
		o.ConsulAddr = addr
		o.ConsulKey = key
	}
}

func WithConfigPath(path string) Option {
	return func(o *Options) { o.ConfigPath = path }
}
`
}

func (c *ConfigSystemGenerator) nacosLoader() string {
	return fmt.Sprintf(`package remote

import (
	"os"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"%s/pkg/config"
)

type NacosLoader struct {
	client config_client.IConfigClient
	dataID string
	group  string
}

func NewNacosLoader(addr, dataID, group, ns string) config.Loader {
	if addr == "" { addr = getEnv("NACOS_ADDR", "127.0.0.1:8848") }
	if dataID == "" { dataID = getEnv("NACOS_DATAID", "application.yaml") }
	if group == "" { group = getEnv("NACOS_GROUP", "DEFAULT_GROUP") }
	if ns == "" { ns = getEnv("NACOS_NS", "public") }
	
	loader := &NacosLoader{dataID: dataID, group: group}
	sc := []constant.ServerConfig{*constant.NewServerConfig(addr, 8848)}
	cc := &constant.ClientConfig{NamespaceId: ns, TimeoutMs: 5000, NotLoadCacheAtStart: true}
	client, _ := clients.NewConfigClient(vo.NacosClientParam{ClientConfig: cc, ServerConfigs: sc})
	loader.client = client
	return loader
}

func (l *NacosLoader) Name() string { return "nacos" }
func (l *NacosLoader) Priority() config.Priority { return config.PriorityRemote }

func (l *NacosLoader) Load(key string) (interface{}, bool) {
	if l.client == nil { return nil, false }
	value, err := l.client.GetConfig(vo.ConfigParam{DataId: l.dataID, Group: l.group})
	if err != nil { return nil, false }
	return value, true
}

func (l *NacosLoader) Watch(key string, cb func(string, interface{})) error {
	return l.client.ListenConfig(vo.ConfigParam{DataId: l.dataID, Group: l.group, OnChange: func(_, _, _, data string) { cb(l.dataID, data) }})
}

func (l *NacosLoader) Close() error { return nil }

func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
`, c.serviceName)
}

func (c *ConfigSystemGenerator) apolloLoader() string {
	return fmt.Sprintf(`package remote

import (
	"os"
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"%s/pkg/config"
)

type ApolloLoader struct {
	client agollo.Client
	ns     string
}

func NewApolloLoader(addr, appID, ns string) config.Loader {
	if addr == "" { addr = getEnv("APOLLO_ADDR", "http://localhost:8080") }
	if appID == "" { appID = getEnv("APOLLO_APPID", "application") }
	if ns == "" { ns = getEnv("APOLLO_NS", "application") }
	
	loader := &ApolloLoader{ns: ns}
	cfg := &config.AppConfig{AppID: appID, Cluster: "default", NamespaceName: ns, IP: addr}
	client, _ := agollo.StartWithConfig(func() (*config.AppConfig, error) { return cfg, nil })
	loader.client = client
	return loader
}

func (l *ApolloLoader) Name() string { return "apollo" }
func (l *ApolloLoader) Priority() config.Priority { return config.PriorityRemote }

func (l *ApolloLoader) Load(key string) (interface{}, bool) {
	if l.client == nil { return nil, false }
	value := l.client.GetConfig(l.ns).GetValue(key)
	if value == "" { return nil, false }
	return value, true
}

func (l *ApolloLoader) Watch(key string, cb func(string, interface{})) error {
	l.client.ListenToChannel(func(event *agollo.ChangeEvent) {
		for k, change := range event.Changes { if k == key { cb(key, change.NewValue) } }
	})
	return nil
}

func (l *ApolloLoader) Close() error { return nil }

func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
`, c.serviceName)
}

func (c *ConfigSystemGenerator) consulLoader() string {
	return fmt.Sprintf(`package remote

import (
	"os"
	"encoding/json"
	consul "github.com/hashicorp/consul/api"
	"%s/pkg/config"
)

type ConsulLoader struct {
	client *consul.Client
	key    string
}

func NewConsulLoader(addr, key string) config.Loader {
	if addr == "" { addr = getEnv("CONSUL_ADDR", "127.0.0.1:8500") }
	if key == "" { key = getEnv("CONSUL_KEY", "config/application") }
	
	loader := &ConsulLoader{key: key}
	cfg := consul.DefaultConfig()
	cfg.Address = addr
	client, _ := consul.NewClient(cfg)
	loader.client = client
	return loader
}

func (l *ConsulLoader) Name() string { return "consul" }
func (l *ConsulLoader) Priority() config.Priority { return config.PriorityRemote }

func (l *ConsulLoader) Load(key string) (interface{}, bool) {
	if l.client == nil { return nil, false }
	pair, _, _ := l.client.KV().Get(l.key, nil)
	if pair == nil { return nil, false }
	var data map[string]interface{}
	json.Unmarshal(pair.Value, &data)
	if v, ok := data[key]; ok { return v, true }
	return string(pair.Value), true
}

func (l *ConsulLoader) Watch(key string, cb func(string, interface{})) error {
	go func() {
		index := uint64(0)
		for {
			pair, meta, _ := l.client.KV().Get(l.key, &consul.QueryOptions{WaitIndex: index})
			index = meta.LastIndex
			if pair != nil { cb(l.key, string(pair.Value)) }
		}
	}()
	return nil
}

func (l *ConsulLoader) Close() error { return nil }

func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
`, c.serviceName)
}

func (c *ConfigSystemGenerator) viperLoader() string {
	return fmt.Sprintf(`package local

import (
	"os"
	"strings"
	"github.com/spf13/viper"
	"%s/pkg/config"
)

type ViperLoader struct {
	viper *viper.Viper
}

func NewViperLoader(configPath string) config.Loader {
	loader := &ViperLoader{viper: viper.New()}
	if configPath == "" { configPath = getEnv("CONFIG_PATH", "configs/config.yaml") }
	loader.viper.SetConfigFile(configPath)
	loader.viper.SetConfigType("yaml")
	loader.viper.SetEnvPrefix("APP")
	loader.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	loader.viper.AutomaticEnv()
	loader.viper.WatchConfig()
	loader.viper.ReadInConfig()
	return loader
}

func (l *ViperLoader) Name() string { return "viper" }
func (l *ViperLoader) Priority() config.Priority { return config.PriorityFile }

func (l *ViperLoader) Load(key string) (interface{}, bool) {
	value := l.viper.Get(key)
	if value == nil { return nil, false }
	return value, true
}

func (l *ViperLoader) Watch(key string, cb func(string, interface{})) error {
	l.viper.OnConfigChange(func(e viper.ConfigChange) { cb(e.Key(), l.viper.Get(e.Key())) })
	return nil
}

func (l *ViperLoader) Close() error { return nil }

func getEnv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
`, c.serviceName)
}

func (c *ConfigSystemGenerator) envLoader() string {
	return fmt.Sprintf(`package local

import (
	"os"
	"strconv"
	"strings"
	"%s/pkg/config"
)

type EnvLoader struct{}

func NewEnvLoader() config.Loader { return &EnvLoader{} }
func (l *EnvLoader) Name() string { return "env" }
func (l *EnvLoader) Priority() config.Priority { return config.PriorityEnv }

func (l *EnvLoader) Load(key string) (interface{}, bool) {
	envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	value := os.Getenv(envKey)
	if value == "" { return nil, false }
	if b, err := strconv.ParseBool(value); err == nil { return b, true }
	if i, err := strconv.ParseInt(value, 10, 64); err == nil { return i, true }
	if f, err := strconv.ParseFloat(value, 64); err == nil { return f, true }
	return value, true
}

func (l *EnvLoader) Watch(key string, cb func(string, interface{})) error { return nil }
func (l *EnvLoader) Close() error { return nil }
`, c.serviceName)
}

func (c *ConfigSystemGenerator) configYAML() string {
	return fmt.Sprintf(`# Application Configuration
app:
  name: ${{APP_NAME:-%s}}
  env: ${{GO_ENV:-local}}
  debug: ${{APP_DEBUG:-false}}

server:
  address: ${{SERVER_ADDR:-:8080}}

database:
  driver: ${{DB_DRIVER:-mysql}}
  host: ${{DB_HOST:-localhost}}
  port: ${{DB_PORT:-3306}}
  user: ${{DB_USER:-root}}
  password: ${{DB_PASSWORD:-}}
  database: ${{DB_NAME:-%s}}

redis:
  addr: ${{REDIS_ADDR:-localhost:6379}}
  password: ${{REDIS_PASSWORD:-}}

log:
  level: ${{LOG_LEVEL:-debug}}
  format: ${{LOG_FORMAT:-json}}

nacos:
  enabled: ${{NACOS_ENABLED:-false}}
  addr: ${{NACOS_ADDR:-127.0.0.1:8848}}
  dataid: ${{NACOS_DATAID:-application.yaml}}
  group: ${{NACOS_GROUP:-DEFAULT_GROUP}}
`, c.serviceName, c.serviceName)
}

func (c *ConfigSystemGenerator) localYAML() string {
	return `# Local Configuration (gitignored)
app:
  debug: true

database:
  host: localhost
  password: local_password

log:
  level: debug
  format: text
`
}

func (c *ConfigSystemGenerator) envExample() string {
	return fmt.Sprintf(`# Environment Variables
APP_NAME=%s
GO_ENV=local
APP_DEBUG=true

SERVER_ADDR=:8080

DB_DRIVER=mysql
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=%s

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

LOG_LEVEL=debug
LOG_FORMAT=json

NACOS_ENABLED=false
NACOS_ADDR=127.0.0.1:8848
NACOS_DATAID=application.yaml
NACOS_GROUP=DEFAULT_GROUP
`, c.serviceName, c.serviceName)
}
