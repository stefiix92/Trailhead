package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is loaded from environment variables. Single-binary / container friendly.
type Config struct {
	// Loki base URL, e.g. https://loki.example.com (no trailing path).
	LokiBaseURL     string
	LokiOrgID       string
	LokiBearer      string
	MarkersFile     string
	MaxSessionLines int
	DevTools        bool
}

// Default max stored line records per MCP session.
const DefaultMaxSessionLines = 100_000

// FromEnv returns configuration from the process environment.
func FromEnv() (Config, error) {
	c := Config{
		LokiBaseURL:     strings.TrimSpace(os.Getenv("TRAILHEAD_LOKI_URL")),
		LokiOrgID:       strings.TrimSpace(os.Getenv("TRAILHEAD_LOKI_TENANT")),
		LokiBearer:      strings.TrimSpace(os.Getenv("TRAILHEAD_LOKI_BEARER_TOKEN")),
		MarkersFile:     strings.TrimSpace(os.Getenv("TRAILHEAD_MARKERS_FILE")),
		DevTools:        strings.TrimSpace(os.Getenv("TRAILHEAD_DEV_TOOLS")) == "1",
		MaxSessionLines: DefaultMaxSessionLines,
	}
	if s := strings.TrimSpace(os.Getenv("TRAILHEAD_MAX_SESSION_LINES")); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return c, fmt.Errorf("TRAILHEAD_MAX_SESSION_LINES: invalid value %q", s)
		}
		c.MaxSessionLines = n
	}
	return c, nil
}

// LokiConfigured reports whether Loki search tools can be used.
func (c Config) LokiConfigured() bool {
	return c.LokiBaseURL != ""
}
