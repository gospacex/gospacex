package scriptcenter

import (
	"strings"

	"github.com/gospacex/gpx/internal/config"
)

func normalizeDBTypes(cfg *config.ProjectConfig) []string {
	if cfg == nil {
		return []string{"mysql"}
	}

	dbTypes := normalizeStringList(cfg.DB)
	if len(dbTypes) == 0 {
		dbTypes = []string{"mysql"}
	}

	normalized := make([]string, 0, len(dbTypes)+1)
	seen := make(map[string]bool, len(dbTypes)+1)
	for _, dbType := range dbTypes {
		canonical := canonicalDBType(dbType)
		if canonical == "" || seen[canonical] {
			continue
		}
		seen[canonical] = true
		normalized = append(normalized, canonical)
	}

	if cfg.MySQLTable != "" && !seen["mysql"] {
		normalized = append([]string{"mysql"}, normalized...)
	}

	if len(normalized) == 0 {
		return []string{"mysql"}
	}
	return normalized
}

func normalizeMQTypes(raw string) []string {
	types := normalizeCommaList(raw)
	normalized := make([]string, 0, len(types))
	seen := make(map[string]bool, len(types))
	for _, mqType := range types {
		canonical := canonicalMQType(mqType)
		if canonical == "" || seen[canonical] {
			continue
		}
		seen[canonical] = true
		normalized = append(normalized, canonical)
	}
	return normalized
}

func normalizeStringList(values []string) []string {
	var normalized []string
	for _, value := range values {
		normalized = append(normalized, normalizeCommaList(value)...)
	}
	return normalized
}

func normalizeCommaList(value string) []string {
	var normalized []string
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		normalized = append(normalized, part)
	}
	return normalized
}

func canonicalDBType(dbType string) string {
	switch strings.ToLower(strings.TrimSpace(dbType)) {
	case "postgres", "pg":
		return "postgresql"
	case "sqlite3":
		return "sqlite"
	case "mongo":
		return "mongodb"
	case "es":
		return "elasticsearch"
	default:
		return strings.ToLower(strings.TrimSpace(dbType))
	}
}

func canonicalMQType(mqType string) string {
	switch strings.ToLower(strings.TrimSpace(mqType)) {
	case "rabbit":
		return "rabbitmq"
	case "mongo":
		return "mongodb"
	case "es":
		return "elasticsearch"
	default:
		return strings.ToLower(strings.TrimSpace(mqType))
	}
}

func containsType(types []string, target string) bool {
	for _, item := range types {
		if item == target {
			return true
		}
	}
	return false
}
