package scene

import (
	"go.uber.org/zap/zapcore"
)

type LevelManagerInterface interface {
	SetLevel(scene string, level zapcore.Level)
	GetLevel(scene string) zapcore.Level
}
