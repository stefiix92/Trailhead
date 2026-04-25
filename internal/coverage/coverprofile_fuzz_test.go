package coverage

import "testing"

func FuzzParseCoverProfile(f *testing.F) {
	f.Add([]byte("mode: set\nfoo.go:1.1,1.2 1 1\n"))
	f.Add([]byte("mode: atomic\n"))
	f.Add([]byte(""))
	f.Add([]byte("not a coverprofile"))

	f.Fuzz(func(t *testing.T, b []byte) {
		sum, err := ParseCoverProfile(b, ParseOptions{TopFiles: 10})
		if err != nil {
			return
		}
		if sum.Mode == "" {
			t.Fatalf("expected non-empty mode on success")
		}
		if sum.TotalStatements < 0 || sum.CoveredStatements < 0 {
			t.Fatalf("negative statement counts: %+v", sum)
		}
		if sum.CoveredStatements > sum.TotalStatements {
			t.Fatalf("covered > total: %+v", sum)
		}
	})
}

