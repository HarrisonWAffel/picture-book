package sync

import (
	"context"
	mutex "sync"

	"github.com/go-co-op/gocron"
)

// SyncerPool holds a reference to each running Syncer
// and ensures that multiple Syncers cannot be started for
// a single registry. It also sets up global contexts for
// graceful termination. There can only ever be 1 SyncerPool.
type SyncerPool struct {
	// Syncers is a mapping between a remote registry
	// and its Syncer.
	Syncers map[string]*Syncer
	mutex.RWMutex
	context.Context
	CronJobScheduler *gocron.Scheduler
}
