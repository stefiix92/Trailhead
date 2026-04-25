package backends

import (
	"context"
	"testing"
)

func TestJournaldBackend_Search_rejectsBadIdentifier(t *testing.T) {
	b := &JournaldBackend{Identifier: "bad=eq"}
	_, err := b.Search(context.Background(), SearchQuery{Limit: 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

