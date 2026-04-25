package config

// ProjectConfig 项目配置结构体
type ProjectConfig struct {
	// 基础配置
	ProjectType  string
	OutputDir    string
	ProjectName  string
	ModuleName   string
	GoModuleName string
	ServiceName  string
	Env          string

	// 微服务配置
	Style      string   // standard, ddd, istio, clean-arch, service-mesh
	IDL        string   // protobuf, thrift
	Protocol   string   // gRPC protocol: grpc (default), kitex
	RPC        string   // kitex
	WithLayers []string // 额外生成的层级: api, rpc, bff

	// 数据存储配置
	DB    []string // mysql, postgresql, redis, mongodb, elasticsearch, etcd
	ORM   string   // gorm, xorm
	Cache string   // redis

	// 中间件配置
	Registry       string // etcd, consul
	Config         string // nacos, apollo, consul
	Trace          string // jaeger
	MQ             string // kafka, rabbitmq, rocketmq
	MQType         string // basic, ordered, delayed, broadcast, pubsub, transaction
	NacosEnabled   bool
	ConsulEnabled  bool
	EtcdEnabled    bool
	SwaggerEnabled bool

	// 分布式事务配置
	DTMEnabled bool
	DTMServer  string
	DTMMode    string // saga, tcc, msg, workflow

	// MySQL 连接配置（用于从 MySQL 提取表结构生成 ES CRUD）
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	MySQLTable    string

	// ES 配置
	ESHost     string
	ESPort     string
	ESUser     string
	ESPassword string
	ESIndex    string

	// Redis 配置
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPrefix   string

	// 对象存储配置
	ObjectDB     string // oss, minio, rustfs
	OSSEndpoint       string
	OSSBucket        string
	OSSAccessKeyID   string
	OSSAccessKeySecret string
	MinioEndpoint    string
	MinioBucket     string
	MinioAccessKey  string
	MinioSecretKey  string
	RustfsPath      string

	// 向量数据库配置
	ZVecHost       string
	ZVecPort       string
	ZVecCollection string

	// MQ 配置
	MQBrokers string
	MQURL     string
	MQNamesrv string
	MQGroupID string

	// pkg 组件开关
	PkgSnowflake bool // 是否生成 pkg/snowflake 组件
}

// NewProjectConfig 创建默认配置
func NewProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		RPC: "kitex",
	}
}
