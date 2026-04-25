package backends

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLokiBackend_Search_parsesStreams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Fatalf("path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": [
					{
						"stream": {"app": "a"},
						"values": [
							["1609459200000000000", "one"],
							["1609459201000000000", "two"]
						]
					}
				]
			}
		}`))
	}))
	defer ts.Close()

	b := LokiBackend{BaseURL: ts.URL}
	lines, err := b.Search(context.Background(), SearchQuery{
		LokiLogQL: `up`,
		Limit:     10,
		Start:     time.Unix(0, 0).UTC(),
		End:       time.Unix(2000000000, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d", len(lines))
	}
	if lines[0].Message != "one" {
		t.Fatalf("msg0 %q", lines[0].Message)
	}
}
