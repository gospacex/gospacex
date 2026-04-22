package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Registry RegistryConfig `yaml:"registry"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	User string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RegistryConfig struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
