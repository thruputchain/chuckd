package storage

import "context"

type Storage struct {
}

func NewStorage(ctx context.Context, path string) *Storage {
	return &Storage{}
}
