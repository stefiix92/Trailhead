package query

import (
	"fmt"
	"strings"
)

// buildLokiErrorLogQL produces a stream selector and pipeline for error-like logs.
// Either service (label key fixed to "service") or raw streamSelector must be set.
// extraContains is an optional case-sensitive LogQL `|=` filter (query string).
func buildLokiErrorLogQL(service, streamSelector, extraContains string) (string, error) {
	selector := strings.TrimSpace(streamSelector)
	if selector == "" {
		svc := strings.TrimSpace(service)
		if svc == "" {
			return "", fmt.Errorf("loki: set service, streamSelector, or lokiLogql on the tool call")
		}
		selector = fmt.Sprintf(`{service=%q}`, svc)
	}
	// Loki/LogQL: filter error-shaped lines, then optional grep.
	q := fmt.Sprintf(`%s |~ "(?i)(error|exception|panic|fatal)"`, selector)
	extra := strings.TrimSpace(extraContains)
	if extra != "" {
		q = fmt.Sprintf(`%s |= %q`, q, extra)
	}
	return q, nil
}

// buildLokiSearchLogQL is a looser search for the search tool: stream + optional line contains.
func buildLokiSearchLogQL(service, streamSelector, contains string) (string, error) {
	selector := strings.TrimSpace(streamSelector)
	if selector == "" {
		svc := strings.TrimSpace(service)
		if svc == "" {
			return "", fmt.Errorf("loki: set service, streamSelector, or lokiLogql")
		}
		selector = fmt.Sprintf(`{service=%q}`, svc)
	}
	c := strings.TrimSpace(contains)
	if c == "" {
		return selector, nil
	}
	return fmt.Sprintf(`%s |= %q`, selector, c), nil
}
