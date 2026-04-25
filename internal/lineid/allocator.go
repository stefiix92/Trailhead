package lineid

import (
	"fmt"
	"sync/atomic"
)

// Allocator generates session-scoped, ergonomic line IDs like `log:4021`.
// It is intentionally simple in the skeleton; v0 will likely include
// source-prefixed IDs (e.g. loki:123, docker:98) to avoid collisions.
type Allocator struct {
	prefix string
	seq    uint64
}

func NewAllocator(prefix string) *Allocator {
	return &Allocator{prefix: prefix, seq: 4000}
}

func (a *Allocator) Next() string {
	n := atomic.AddUint64(&a.seq, 1)
	return fmt.Sprintf("%s:%d", a.prefix, n)
}
