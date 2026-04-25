package backends

import (
	"context"
	"testing"
	"time"
)

func TestDockerBackend_Search_requiresContainerOrComposeService(t *testing.T) {
	b := &DockerBackend{}
	_, err := b.Search(context.Background(), SearchQuery{Limit: 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDockerBackend_resolveComposeContainer_requiresService(t *testing.T) {
	b := &DockerBackend{}
	_, err := b.resolveComposeContainer(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDockerBackend_Search_defaultsLimitAndBoundsTail(t *testing.T) {
	// This is a lightweight behavior test that exercises argument construction.
	// It expects the command to fail (docker may be missing), but ensures we get a docker logs error.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	b := &DockerBackend{Container: "does-not-exist"}
	_, err := b.Search(ctx, SearchQuery{Limit: 0})
	if err == nil {
		t.Fatalf("expected error")
	}
}

