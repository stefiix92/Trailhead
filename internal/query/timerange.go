package query

import (
	"fmt"
	"time"
)

// parseTimeWindow resolves optional RFC3339 start/end, optional since duration, and optional until (RFC3339) as end.
// If end is zero, it defaults to now. If start is not set, it is end - (since or 1h).
func parseTimeWindow(since, startRFC3339, endRFC3339, untilRFC3339 string) (start, end time.Time, err error) {
	now := time.Now().UTC()
	end = now
	if endRFC3339 != "" {
		end, err = time.Parse(time.RFC3339, endRFC3339)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("end: %w", err)
		}
	} else if untilRFC3339 != "" {
		end, err = time.Parse(time.RFC3339, untilRFC3339)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("until: %w", err)
		}
	}
	if startRFC3339 != "" {
		start, err = time.Parse(time.RFC3339, startRFC3339)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("start: %w", err)
		}
		if !end.After(start) {
			return time.Time{}, time.Time{}, fmt.Errorf("time range: end must be after start")
		}
		return start, end, nil
	}
	d := time.Hour
	if since != "" {
		pd, perr := time.ParseDuration(since)
		if perr != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("since: %w", perr)
		}
		if pd <= 0 {
			return time.Time{}, time.Time{}, fmt.Errorf("since must be positive")
		}
		d = pd
	}
	start = end.Add(-d)
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("time range: end must be after start")
	}
	return start, end, nil
}
