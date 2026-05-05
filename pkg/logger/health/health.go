package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type HealthStatus struct {
	Status          string             `json:"status"`
	BufferUsage     map[string]float64 `json:"buffer_usage"`
	MQStatus        string             `json:"mq_status"`
	DegradedReasons []string          `json:"degraded_reasons,omitempty"`
}

type Producer interface {
	Healthy() bool
}

var (
	mqProducer   atomic.Value
	bufferStatus atomic.Value
)

func RegisterMQProducer(p Producer) {
	mqProducer.Store(p)
}

func UpdateBufferStatus(scene string, usage float64) {
	status := bufferStatus.Load()
	if status == nil {
		status = make(map[string]float64)
	}
	m := status.(map[string]float64)
	m[scene] = usage
	bufferStatus.Store(m)
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:      "healthy",
		BufferUsage: make(map[string]float64),
		MQStatus:   "healthy",
	}

	bufferMap := bufferStatus.Load()
	if bufferMap != nil {
		status.BufferUsage = bufferMap.(map[string]float64)
		for scene, usage := range status.BufferUsage {
			if usage > 0.9 {
				status.Status = "degraded"
				status.DegradedReasons = append(status.DegradedReasons, "buffer_usage for "+scene+" exceeds 0.9")
			}
		}
	}

	producer := mqProducer.Load()
	if producer != nil {
		if p, ok := producer.(Producer); ok {
			if !p.Healthy() {
				status.MQStatus = "unhealthy"
				status.Status = "degraded"
				status.DegradedReasons = append(status.DegradedReasons, "mq producer is unhealthy")
			}
		}
	}

	statusCode := http.StatusOK
	if status.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}
