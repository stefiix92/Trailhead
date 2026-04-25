// Package cluster groups log messages with TF–IDF cosine similarity (greedy, capped clusters).
package cluster

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	reNumber  = regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	reHexLong = regexp.MustCompile(`\b[0-9a-fA-F]{8,}\b`)
	reUUID    = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// Group represents one cluster of line indices in the input slice.
type Group struct {
	ID          string
	Size        int
	Signature   string
	MemberIndex []int
	Score       float64
}

// ClusterTFIDF groups messages into up to maxClusters using TF–IDF + cosine similarity
// to a per-cluster representative (first member).
func ClusterTFIDF(messages []string, maxClusters int) []Group {
	if maxClusters <= 0 {
		maxClusters = 12
	}
	if len(messages) == 0 {
		return nil
	}
	if len(messages) < 2 {
		return []Group{{
			ID:          "c1",
			Size:        1,
			Signature:   signature(messages[0]),
			MemberIndex: []int{0},
			Score:       1,
		}}
	}

	docs := make([][]string, len(messages))
	for i, m := range messages {
		docs[i] = tokenize(m)
	}
	vecs := make([]map[string]float64, len(messages))
	df := docFreq(docs)
	for i := range messages {
		vecs[i] = tfidfVector(docs[i], df, len(messages))
	}

	// Clusters: list of line indices
	var clusters [][]int
	const simThreshold = 0.38

	for i := 0; i < len(messages); i++ {
		bestJ := -1
		bestS := 0.0
		for j := range clusters {
			rep := clusters[j][0]
			s := cosine(vecs[i], vecs[rep])
			if s > bestS {
				bestS = s
				bestJ = j
			}
		}
		if bestJ >= 0 && bestS >= simThreshold {
			clusters[bestJ] = append(clusters[bestJ], i)
			continue
		}
		if len(clusters) < maxClusters {
			clusters = append(clusters, []int{i})
			continue
		}
		// merge into best existing cluster
		if bestJ < 0 {
			bestJ = 0
		}
		clusters[bestJ] = append(clusters[bestJ], i)
	}

	groups := make([]Group, 0, len(clusters))
	for i, idxs := range clusters {
		size := len(idxs)
		sig := signature(messages[idxs[0]])
		// total similarity mass (cohesion proxy)
		scr := 0.0
		if size > 1 {
			rep := idxs[0]
			for k := 1; k < size; k++ {
				scr += cosine(vecs[idxs[k]], vecs[rep])
			}
		} else {
			scr = 1
		}
		groups = append(groups, Group{
			ID:          "c" + strconv.Itoa(i+1),
			Size:        size,
			Signature:   sig,
			MemberIndex: append([]int(nil), idxs...),
			Score:       scr,
		})
	}

	sort.Slice(groups, func(i, j int) bool { return groups[i].Size > groups[j].Size })
	// renumber IDs c1, c2 by sorted size
	for i := range groups {
		groups[i].ID = "c" + strconv.Itoa(i+1)
	}
	return groups
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	s = reUUID.ReplaceAllString(s, "uuid")
	s = reHexLong.ReplaceAllString(s, "hex")
	s = reNumber.ReplaceAllString(s, "n")
	fields := strings.Fields(s)
	// unigrams + a few bigrams
	var out []string
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		out = append(out, f)
	}
	for i := 0; i+1 < len(fields); i++ {
		if len(fields[i]) < 2 || len(fields[i+1]) < 2 {
			continue
		}
		out = append(out, fields[i]+"_"+fields[i+1])
	}
	if len(out) == 0 {
		return []string{"_empty_"}
	}
	return out
}

func docFreq(docs [][]string) map[string]int {
	df := make(map[string]int)
	seen := make(map[int]map[string]struct{})
	for i, d := range docs {
		seen[i] = make(map[string]struct{})
		for _, t := range d {
			seen[i][t] = struct{}{}
		}
	}
	for i := range docs {
		for t := range seen[i] {
			df[t]++
		}
	}
	return df
}

func tfidfVector(tokens []string, df map[string]int, nDocs int) map[string]float64 {
	if len(tokens) == 0 {
		return map[string]float64{"_empty_": 1}
	}
	tf := make(map[string]int)
	for _, t := range tokens {
		tf[t]++
	}
	v := make(map[string]float64)
	for t, c := range tf {
		idf := math.Log(float64(1+nDocs) / float64(1+df[t]))
		if idf < 0 {
			idf = 0
		}
		v[t] = float64(c) * idf
	}
	norm := l2(v)
	if norm > 0 {
		for t := range v {
			v[t] /= norm
		}
	}
	return v
}

func l2(m map[string]float64) float64 {
	s := 0.0
	for _, v := range m {
		s += v * v
	}
	if s == 0 {
		return 0
	}
	return math.Sqrt(s)
}

func cosine(a, b map[string]float64) float64 {
	var num float64
	for t, va := range a {
		if vb, ok := b[t]; ok {
			num += va * vb
		}
	}
	return num
}

func signature(msg string) string {
	s := reUUID.ReplaceAllString(msg, "<uuid>")
	s = reNumber.ReplaceAllString(s, "<n>")
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
