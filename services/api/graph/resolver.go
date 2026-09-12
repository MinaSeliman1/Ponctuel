package graph

import (
	"context"
	"time"

	"ponctuel/services/ingester/store"
)

type Resolver struct {
	repository      store.Repository
	mode            string
	freshnessWindow time.Duration
}

func NewResolver(repository store.Repository, mode string, freshnessWindow time.Duration) *Resolver {
	if freshnessWindow <= 0 {
		freshnessWindow = 2 * time.Minute
	}
	return &Resolver{repository: repository, mode: mode, freshnessWindow: freshnessWindow}
}

type snapshotReader interface {
	LatestSnapshotAt(ctx context.Context) (time.Time, error)
}
