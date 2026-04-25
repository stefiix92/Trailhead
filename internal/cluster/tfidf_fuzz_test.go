package cluster

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzClusterTFIDF_invariants(f *testing.F) {
	f.Add("error timeout A\nerror timeout B\nfatal db down")
	f.Add("a\nb")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		msgs := strings.Split(input, "\n")
		if len(msgs) > 200 {
			msgs = msgs[:200]
		}
		groups := ClusterTFIDF(msgs, 12)
		if len(msgs) == 0 {
			if len(groups) != 0 {
				t.Fatalf("expected no groups")
			}
			return
		}
		seen := make([]bool, len(msgs))
		total := 0
		ids := map[string]struct{}{}
		for _, g := range groups {
			if g.ID == "" {
				t.Fatalf("empty group id")
			}
			if _, ok := ids[g.ID]; ok {
				t.Fatalf("duplicate group id %q", g.ID)
			}
			ids[g.ID] = struct{}{}

			if g.Size != len(g.MemberIndex) {
				t.Fatalf("size mismatch: %+v", g)
			}
			if g.Size <= 0 {
				t.Fatalf("non-positive size: %+v", g)
			}
			for _, idx := range g.MemberIndex {
				if idx < 0 || idx >= len(msgs) {
					t.Fatalf("index out of range: %d", idx)
				}
				if seen[idx] {
					t.Fatalf("duplicate member index %d", idx)
				}
				seen[idx] = true
				total++
			}
		}
		if total != len(msgs) {
			missing := 0
			for i := range seen {
				if !seen[i] {
					missing++
				}
			}
			t.Fatalf("not all messages covered: total=%d len=%d missing=%d", total, len(msgs), missing)
		}
		if len(groups) > 12 {
			t.Fatalf("exceeded maxClusters: %d", len(groups))
		}
		// IDs should be c1..cN after sorting/renumbering.
		for i := range groups {
			want := fmt.Sprintf("c%d", i+1)
			if groups[i].ID != want {
				t.Fatalf("unexpected id at %d: got %q want %q", i, groups[i].ID, want)
			}
		}
	})
}

