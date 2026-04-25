package coverage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type ParseOptions struct {
	// TopFiles bounds the per-file breakdown (highest statement count first).
	TopFiles int
}

type FileCoverage struct {
	Path             string  `json:"path"`
	TotalStatements  int64   `json:"total_statements"`
	CoveredStatements int64  `json:"covered_statements"`
	Percent          float64 `json:"percent"`
}

type CoverProfileSummary struct {
	Mode              string        `json:"mode"`
	TotalStatements   int64         `json:"total_statements"`
	CoveredStatements int64         `json:"covered_statements"`
	TotalPercent      float64       `json:"total_percent"`
	TopFiles          []FileCoverage `json:"top_files,omitempty"`
}

var ErrInvalidCoverProfile = errors.New("invalid coverprofile")

// ParseCoverProfile parses Go's coverprofile format produced by:
//   go test -coverprofile=...
//
// It returns a bounded summary suitable for structured tool output.
func ParseCoverProfile(profile []byte, opts ParseOptions) (CoverProfileSummary, error) {
	sc := bufio.NewScanner(bytes.NewReader(profile))

	if !sc.Scan() {
		return CoverProfileSummary{}, fmt.Errorf("%w: empty", ErrInvalidCoverProfile)
	}

	first := strings.TrimSpace(sc.Text())
	if !strings.HasPrefix(first, "mode:") {
		return CoverProfileSummary{}, fmt.Errorf("%w: missing mode header", ErrInvalidCoverProfile)
	}
	mode := strings.TrimSpace(strings.TrimPrefix(first, "mode:"))
	if mode == "" {
		return CoverProfileSummary{}, fmt.Errorf("%w: empty mode", ErrInvalidCoverProfile)
	}

	type agg struct{ total, covered int64 }
	perFile := map[string]*agg{}
	var total, covered int64

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		// Format: file:startLine.startCol,endLine.endCol numStatements count
		// Split from the right to tolerate ':' in Windows paths in the future.
		parts := strings.Fields(line)
		if len(parts) != 3 {
			return CoverProfileSummary{}, fmt.Errorf("%w: bad line %q", ErrInvalidCoverProfile, line)
		}

		fileAndRange := parts[0]
		numStmtsStr := parts[1]
		countStr := parts[2]

		// fileAndRange: <file>:<range>
		colon := strings.LastIndex(fileAndRange, ":")
		if colon <= 0 || colon == len(fileAndRange)-1 {
			return CoverProfileSummary{}, fmt.Errorf("%w: bad file/range %q", ErrInvalidCoverProfile, fileAndRange)
		}
		file := fileAndRange[:colon]

		numStmts, err := strconv.ParseInt(numStmtsStr, 10, 64)
		if err != nil || numStmts < 0 {
			return CoverProfileSummary{}, fmt.Errorf("%w: bad statements %q", ErrInvalidCoverProfile, numStmtsStr)
		}
		count, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil || count < 0 {
			return CoverProfileSummary{}, fmt.Errorf("%w: bad count %q", ErrInvalidCoverProfile, countStr)
		}

		total += numStmts
		if count > 0 {
			covered += numStmts
		}

		a := perFile[file]
		if a == nil {
			a = &agg{}
			perFile[file] = a
		}
		a.total += numStmts
		if count > 0 {
			a.covered += numStmts
		}
	}
	if err := sc.Err(); err != nil {
		return CoverProfileSummary{}, err
	}

	sum := CoverProfileSummary{
		Mode:              mode,
		TotalStatements:   total,
		CoveredStatements: covered,
		TotalPercent:      percent(covered, total),
	}

	if opts.TopFiles > 0 && len(perFile) > 0 {
		files := make([]FileCoverage, 0, len(perFile))
		for path, a := range perFile {
			files = append(files, FileCoverage{
				Path:              path,
				TotalStatements:   a.total,
				CoveredStatements: a.covered,
				Percent:           percent(a.covered, a.total),
			})
		}

		// Sort by total statements descending, then path.
		sort.Slice(files, func(i, j int) bool {
			if files[i].TotalStatements != files[j].TotalStatements {
				return files[i].TotalStatements > files[j].TotalStatements
			}
			return files[i].Path < files[j].Path
		})

		if opts.TopFiles < len(files) {
			files = files[:opts.TopFiles]
		}
		sum.TopFiles = files
	}

	return sum, nil
}

func percent(covered, total int64) float64 {
	if total == 0 {
		return 0
	}
	// Keep one decimal of stability? For now, match tests expecting exact .0.
	return (float64(covered) / float64(total)) * 100.0
}

