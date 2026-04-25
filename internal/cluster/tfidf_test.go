package cluster

import (
	"testing"
)

func TestClusterTFIDF_groups(t *testing.T) {
	msgs := []string{
		"ERROR: connection reset by peer 123",
		"ERROR: connection reset by peer 999",
		"INFO ok",
		"panic: nil pointer in foo",
		"panic: nil pointer in bar",
	}
	g := ClusterTFIDF(msgs, 5)
	if len(g) < 1 {
		t.Fatalf("no clusters")
	}
	n := 0
	for _, c := range g {
		n += c.Size
	}
	if n != len(msgs) {
		t.Fatalf("cover %d != %d", n, len(msgs))
	}
}
