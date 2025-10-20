package storage

import "context"

type DiskStorage struct {
}

func NewStorage(ctx context.Context, path string) *DiskStorage {
	return &DiskStorage{}
}
