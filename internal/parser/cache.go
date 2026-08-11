package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseCacheAnnotation parses a @cache annotation line.
// Format: -- @cache {"ttl": "30s", "name": "UserCache", "tags": ["users"], "dep": ["UpdateUser"]}
func parseCacheAnnotation(line string) (*CacheDef, error) {
	// Strip the "-- @cache " prefix
	raw := strings.TrimPrefix(line, "-- @cache")
	raw = strings.TrimSpace(raw)

	if raw == "" {
		// Bare @cache with no JSON — use defaults
		return &CacheDef{}, nil
	}

	// Parse JSON
	var parsed struct {
		TTL  string   `json:"ttl"`
		Name string   `json:"name"`
		Tags []string `json:"tags"`
		Dep  []string `json:"dep"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("invalid @cache annotation JSON: %w\n  line: %s", err, line)
	}

	return &CacheDef{
		TTL:  parsed.TTL,
		Name: parsed.Name,
		Tags: parsed.Tags,
		Dep:  parsed.Dep,
	}, nil
}
