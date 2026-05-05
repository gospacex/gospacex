package scene

import (
	"time"
)

type Config struct {
	BatchSize          int
	FlushInterval      time.Duration
	Rate               int
	Burst              int
	OverflowAction     string
	LogLevel           string
	LogBody            bool
	SlowThreshold      time.Duration
	VitalBufferSize    int
	VitalSyncTimeout   time.Duration
	VitalFallbackOnFull bool
}
