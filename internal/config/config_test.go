package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectConfig(t *testing.T) {
	cfg := NewProjectConfig()
	assert.Equal(t, "kitex", cfg.RPC)
	assert.Empty(t, cfg.ProjectType)
	assert.Empty(t, cfg.OutputDir)
}

func TestConfigValidator_ValidConfig(t *testing.T) {
	v := NewConfigValidator()

	t.Run("valid microservice", func(t *testing.T) {
		cfg := &ProjectConfig{
			ProjectType: "microservice",
			Style:       "ddd",
			IDL:         "protobuf",
		}
		assert.NoError(t, v.Validate(cfg))
	})

	t.Run("valid monolith", func(t *testing.T) {
		cfg := &ProjectConfig{
			ProjectType: "monolith",
			ORM:         "gorm",
		}
		assert.NoError(t, v.Validate(cfg))
	})

	t.Run("valid script", func(t *testing.T) {
		cfg := &ProjectConfig{
			ProjectType: "script",
			DB:          []string{"mysql"},
		}
		assert.NoError(t, v.Validate(cfg))
	})

	t.Run("valid agent", func(t *testing.T) {
		cfg := &ProjectConfig{
			ProjectType: "agent",
			DB:          []string{"mongodb"},
		}
		assert.NoError(t, v.Validate(cfg))
	})
}

func TestConfigValidator_IstioWithRegistry(t *testing.T) {
	v := NewConfigValidator()

	cfg := &ProjectConfig{
		ProjectType: "microservice",
		Style:       "istio",
		Registry:    "etcd",
	}

	err := v.Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Istio 架构不支持服务注册")
}

func TestConfigValidator_ThriftForNonMicroservice(t *testing.T) {
	v := NewConfigValidator()

	cfg := &ProjectConfig{
		ProjectType: "monolith",
		IDL:         "thrift",
	}

	err := v.Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Thrift IDL 仅支持微服务项目")
}

func TestConfigValidator_MonolithWithIDL(t *testing.T) {
	v := NewConfigValidator()

	cfg := &ProjectConfig{
		ProjectType: "monolith",
		IDL:         "protobuf",
	}

	err := v.Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "单体项目不支持 IDL")
}

func TestConfigValidator_ScriptCenterWithIDL(t *testing.T) {
	v := NewConfigValidator()

	cfg := &ProjectConfig{
		ProjectType: "script",
		IDL:         "protobuf",
	}

	err := v.Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "脚本中心不支持 IDL")
}

func TestConfigValidator_AgentWithIDL(t *testing.T) {
	v := NewConfigValidator()

	cfg := &ProjectConfig{
		ProjectType: "agent",
		IDL:         "protobuf",
	}

	err := v.Validate(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Agent 项目不支持 IDL")
}
