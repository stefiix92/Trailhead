package query

import (
	"strings"
	"testing"
)

func TestBuildLokiErrorLogQL_requiresSelectorOrService(t *testing.T) {
	_, err := buildLokiErrorLogQL("", "", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildLokiErrorLogQL_serviceSelectorAndExtra(t *testing.T) {
	q, err := buildLokiErrorLogQL(" api ", "", `timeout`)
	if err != nil {
		t.Fatalf("buildLokiErrorLogQL: %v", err)
	}
	if !strings.HasPrefix(q, `{service="api"}`) {
		t.Fatalf("unexpected selector: %s", q)
	}
	if !strings.Contains(q, `|~ "(?i)(error|exception|panic|fatal)"`) {
		t.Fatalf("missing error regex stage: %s", q)
	}
	if !strings.Contains(q, `|= "timeout"`) {
		t.Fatalf("missing extra contains stage: %s", q)
	}
}

func TestBuildLokiErrorLogQL_streamSelectorWins(t *testing.T) {
	q, err := buildLokiErrorLogQL("ignored", ` {app="x"} `, "")
	if err != nil {
		t.Fatalf("buildLokiErrorLogQL: %v", err)
	}
	if !strings.HasPrefix(q, `{app="x"}`) {
		t.Fatalf("unexpected selector: %s", q)
	}
	if strings.Contains(q, `{service="ignored"}`) {
		t.Fatalf("service selector should not be used when stream selector is provided: %s", q)
	}
	if !strings.Contains(q, `|~ "(?i)(error|exception|panic|fatal)"`) {
		t.Fatalf("missing error regex stage: %s", q)
	}
}

func TestBuildLokiSearchLogQL_requiresSelectorOrService(t *testing.T) {
	_, err := buildLokiSearchLogQL("", "", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildLokiSearchLogQL_containsOptional(t *testing.T) {
	q, err := buildLokiSearchLogQL("api", "", "")
	if err != nil {
		t.Fatalf("buildLokiSearchLogQL: %v", err)
	}
	if q != `{service="api"}` {
		t.Fatalf("unexpected query: %s", q)
	}

	q2, err := buildLokiSearchLogQL("api", "", "boom")
	if err != nil {
		t.Fatalf("buildLokiSearchLogQL: %v", err)
	}
	if q2 != `{service="api"} |= "boom"` {
		t.Fatalf("unexpected query: %s", q2)
	}
}

func TestBuildLokiSearchLogQL_escapesContains(t *testing.T) {
	q, err := buildLokiSearchLogQL("api", "", `he said "hi"`)
	if err != nil {
		t.Fatalf("buildLokiSearchLogQL: %v", err)
	}
	// keep assertion resilient: just ensure the embedded quote is escaped
	if !strings.Contains(q, `\"hi\"`) {
		t.Fatalf("expected escaped quotes in query: %s", q)
	}
}

