package backends

import (
	"context"
	"errors"
)

var ErrUnsupportedSource = errors.New("unsupported source")

type SearchQuery struct {
	Query string
	Limit int
}

type Line struct {
	Timestamp string
	Message   string
}

type Backend interface {
	Search(ctx context.Context, q SearchQuery) ([]Line, error)
}

