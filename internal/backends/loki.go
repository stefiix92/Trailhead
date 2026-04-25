package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LokiBackend queries Grafana Loki via the log query HTTP API.
type LokiBackend struct {
	BaseURL    string
	HTTPClient *http.Client
	OrgID      string
	Bearer     string
}

func (b *LokiBackend) Search(ctx context.Context, q SearchQuery) ([]Line, error) {
	if b.BaseURL == "" {
		return nil, errors.New("loki base URL is not configured (set TRAILHEAD_LOKI_URL)")
	}
	logQL := strings.TrimSpace(q.LokiLogQL)
	if logQL == "" {
		return nil, errors.New("lokiLogql (or built logql) is required for source=loki")
	}
	if b.HTTPClient == nil {
		b.HTTPClient = http.DefaultClient
	}

	end := q.End
	start := q.Start
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() {
		start = end.Add(-1 * time.Hour)
	}
	if !end.After(start) {
		return nil, errors.New("loki time range: end must be after start")
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}

	u, err := url.Parse(strings.TrimRight(b.BaseURL, "/") + "/loki/api/v1/query_range")
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("query", logQL)
	qs.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	qs.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	qs.Set("limit", strconv.Itoa(limit))
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if b.OrgID != "" {
		req.Header.Set("X-Scope-OrgID", b.OrgID)
	}
	if b.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+b.Bearer)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("loki query_range: HTTP %s: %s", resp.Status, truncateErr(body, 500))
	}

	var wr lokiQueryRangeResponse
	if err := json.Unmarshal(body, &wr); err != nil {
		return nil, fmt.Errorf("loki response: %w", err)
	}
	if wr.Status != "" && wr.Status != "success" {
		return nil, fmt.Errorf("loki error status: %q", wr.Status)
	}
	var out []Line
	for _, s := range wr.Data.Result {
		for _, v := range s.Values {
			if len(v) < 2 {
				continue
			}
			ns, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				continue
			}
			ts := time.Unix(0, ns).UTC()
			out = append(out, Line{
				Timestamp: ts.Format(time.RFC3339Nano),
				Message:   v[1],
			})
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type lokiQueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][2]string       `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func truncateErr(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
