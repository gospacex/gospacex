package conf

type ConfigYaml struct {
	Mode      string        `yaml:"mode"`
	SRV       *SRVConfig    `yaml:"srv"`
	BFF       *BFFConfig    `yaml:"srv"`
	Log       *LogConfig    `yaml:"log"`
	Redis     *RedisConfig  `yaml:"redis"`
	Mysql     *MysqlConfig  `yaml:"mysql"`
	Consul    *ConsulConfig `yaml:"consul"`
	Auth      *Auth         `yaml:"auth"`
	SecretKey string        `json:"secretKey"`
	Chain     *ChainConfig  `json:"chain"`
	Nacos     *NacosConfig  `yaml:"nacos"`
}

var (
	Cfg = &ConfigYaml{
		Mode: "debug",
		SRV: &SRVConfig{
			Name: "srv",
			Host: "0.0.0.0",
			Port: 80,
		},
		Log: &LogConfig{
			Level:         "info",
			MaxSize:       100, // megabytes
			MaxBackups:    5,
			MaxAge:        15, // 15 days
			Compress:      true,
			Path:          "app.log",
			ConsoleEnable: true,
		},
		Auth: &Auth{
			Custom: map[string]string{},
		},
	}
)

type SRVConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type BFFConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Db       int    `yaml:"db"`
	Password string `yaml:"password"`
	MaxIdle  int    `yaml:"maxIdle"`
	PoolSize int    `yaml:"poolSize"`
}

type MysqlConfig struct {
	Ip       string `yaml:"ip"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Db       string `yaml:"db"`
}

type KV struct {
	Key   string
	Value string
}

type AclConfig struct {
	Url           string `yaml:"url"`
	AppId         string `yaml:"appId"`
	SecretKey     string `yaml:"secretKey"`
	ResourceNames []*KV  `yaml:"resourceNames"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
	// MaxSize max size of single file, unit is MB
	MaxSize int `yaml:"maxSize"`
	// MaxBackups max number of backup files
	MaxBackups int `yaml:"maxBackups"`
	// MaxAge max days of backup files, unit is day
	MaxAge int `yaml:"maxAge"`
	// Compress whether compress backup file
	Compress bool `yaml:"compress"`
	// Format
	Format string `yaml:"format"`
	// Console output
	ConsoleEnable bool `yaml:"consoleEnable"`
}

type Auth struct {
	Acl    *AclConfig        `yaml:"acl"`
	Custom map[string]string `yaml:"custom"`
}

type ConsulConfig struct {
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	Services []ServiceInfo `yaml:"services"`
}

type ServiceInfo struct {
	Name string   `json:"name"`
	Host string   `json:"host"`
	Port int      `json:"port"`
	Tags []string `json:"tags"`
}

type ChainConfig struct {
	Enable bool   `yaml:"enable"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Name   string `yaml:"Name"`
}

type NacosConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	NamespaceId string `yaml:"namespaceId"`
	DataId      string `yaml:"dataId"`
	Group       string `yaml:"group"`
}
