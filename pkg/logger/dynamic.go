package logger

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap/zapcore"
)

type LevelHandler struct {
	manager *LevelManager
}

func NewLevelHandler(manager *LevelManager) *LevelHandler {
	return &LevelHandler{manager: manager}
}

func (h *LevelHandler) HandleGetLevel(w http.ResponseWriter, r *http.Request) {
	levels := h.manager.AllLevels()
	resp := make(map[string]string)
	for scene, level := range levels {
		resp[scene] = level.String()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *LevelHandler) HandlePostLevel(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	scene := r.FormValue("scene")
	levelStr := r.FormValue("level")

	if levelStr == "" {
		http.Error(w, "level parameter is required", http.StatusBadRequest)
		return
	}

	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		http.Error(w, "invalid level: "+levelStr, http.StatusBadRequest)
		return
	}

	if scene != "" {
		h.manager.SetLevel(scene, level)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"scene":   scene,
			"level":   level.String(),
		})
		return
	}

	levels := map[string]zapcore.Level{
		"business": level,
		"access":   level,
		"audit":    level,
		"error":    level,
	}
	h.manager.SetLevels(levels)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"level":  level.String(),
	})
}
