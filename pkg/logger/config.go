package logger

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env        string        `yaml:"env"`
	Level      string        `yaml:"level"`
	ServiceName string       `yaml:"service_name"`
	TopicPrefix string       `yaml:"topic_prefix"`

	Sampling  SamplingConfig  `yaml:"sampling"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Rotation  RotationConfig  `yaml:"rotation"`
	Vital     VitalConfig     `yaml:"vital"`
	Storage   StorageConfig   `yaml:"storage"`
	MQ        MQConfig        `yaml:"mq"`
	Tracing   TracingConfig   `yaml:"tracing"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Retention RetentionConfig  `yaml:"retention"`
}

type RetentionConfig struct {
	Audit    int `yaml:"audit"`
	Access   int `yaml:"access"`
	Business int `yaml:"business"`
	Error    int `yaml:"error"`
}

type SamplingConfig struct {
	Warn  float64          `yaml:"warn"`
	Info  InfoSampling     `yaml:"info"`
	Debug float64          `yaml:"debug"`
}

type InfoSampling struct {
	Initial    int           `yaml:"initial"`
	Thereafter int          `yaml:"thereafter"`
	Tick       time.Duration `yaml:"tick"`
}

type RateLimitConfig struct {
	Error RateLimitRule `yaml:"error"`
}

type RateLimitRule struct {
	Rate          int    `yaml:"rate"`
	Burst         int    `yaml:"burst"`
	OverflowAction string `yaml:"overflow_action"`
}

type RotationConfig struct {
	Enabled     bool `yaml:"enabled"`
	MaxAgeDays  int  `yaml:"max_age_days"`
}

type VitalConfig struct {
	BufferSize    int           `yaml:"buffer_size"`
	SyncInterval  time.Duration `yaml:"sync_interval"`
	FsyncTimeout  time.Duration `yaml:"fsync_timeout"`
	FallbackOnFull bool         `yaml:"fallback_on_full"`
}

type StorageConfig struct {
	LogLevel          string        `yaml:"log_level"`
	LogBody           bool          `yaml:"log_body"`
	SlowThreshold    time.Duration `yaml:"log_slow_threshold"`
}

type MQConfig struct {
	Brokers        []string `yaml:"brokers"`
	PartitionCount int32    `yaml:"partition_count"`
	BatchSize     int       `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
}

type TracingConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Endpoint   string `yaml:"endpoint"`
	SampleRate float64 `yaml:"sample_rate"`
}

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		Env:         "dev",
		Level:       "info",
		ServiceName: "unknown",
		TopicPrefix: "app-logs",
		Sampling: SamplingConfig{
			Warn:  1.0,
			Info:  InfoSampling{Initial: 100, Thereafter: 200, Tick: time.Second},
			Debug: 0,
		},
		RateLimit: RateLimitConfig{
			Error: RateLimitRule{Rate: 100, Burst: 200, OverflowAction: "aggregate"},
		},
		Rotation: RotationConfig{
			Enabled:    true,
			MaxAgeDays: 7,
		},
		Vital: VitalConfig{
			BufferSize:     10000,
			SyncInterval:   time.Second,
			FsyncTimeout:   50 * time.Millisecond,
			FallbackOnFull: true,
		},
		Storage: StorageConfig{
			LogLevel:       "debug",
			LogBody:        false,
			SlowThreshold: 100 * time.Millisecond,
		},
		MQ: MQConfig{
			Brokers:        []string{"localhost:9092"},
			PartitionCount: 64,
			BatchSize:      100,
			FlushInterval:  time.Second,
		},
		Tracing: TracingConfig{
			Enabled:    true,
			Endpoint:   "localhost:4318",
			SampleRate: 1.0,
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
		Retention: RetentionConfig{
			Audit:    1095,
			Access:   30,
			Business: 7,
			Error:    7,
		},
	}
}

func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if c.Vital.BufferSize <= 0 {
		return fmt.Errorf("vital.buffer_size must be positive")
	}
	if c.Vital.FsyncTimeout <= 0 {
		return fmt.Errorf("vital.fsync_timeout must be positive")
	}
	if c.MQ.PartitionCount <= 0 {
		return fmt.Errorf("mq.partition_count must be positive")
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return ParseConfig(data)
}

func ParseConfig(data []byte) (*Config, error) {
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}
