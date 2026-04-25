package coverage

import "testing"

func TestParseCoverProfile_totalAndPerFile(t *testing.T) {
	profile := `mode: set
github.com/acme/proj/foo.go:10.1,12.2 1 1
github.com/acme/proj/foo.go:13.1,15.2 1 0
github.com/acme/proj/bar.go:1.1,2.2 2 2
`

	res, err := ParseCoverProfile([]byte(profile), ParseOptions{TopFiles: 10})
	if err != nil {
		t.Fatalf("ParseCoverProfile error: %v", err)
	}

	// Total statements = 1 + 1 + 2 = 4; covered = 1 + 0 + 2 = 3 => 75%
	if res.TotalStatements != 4 {
		t.Fatalf("TotalStatements: want 4, got %d", res.TotalStatements)
	}
	if res.CoveredStatements != 3 {
		t.Fatalf("CoveredStatements: want 3, got %d", res.CoveredStatements)
	}
	if res.TotalPercent != 75.0 {
		t.Fatalf("TotalPercent: want 75.0, got %v", res.TotalPercent)
	}

	if len(res.TopFiles) != 2 {
		t.Fatalf("TopFiles len: want 2, got %d", len(res.TopFiles))
	}

	// foo.go total=2 covered=1 => 50%
	var foo *FileCoverage
	for i := range res.TopFiles {
		if res.TopFiles[i].Path == "github.com/acme/proj/foo.go" {
			foo = &res.TopFiles[i]
			break
		}
	}
	if foo == nil {
		t.Fatalf("expected foo.go entry, got %v", res.TopFiles)
	}
	if foo.Percent != 50.0 {
		t.Fatalf("foo.Percent: want 50.0, got %v", foo.Percent)
	}
}

func TestParseCoverProfile_rejectsMissingMode(t *testing.T) {
	_, err := ParseCoverProfile([]byte("nope\n"), ParseOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

