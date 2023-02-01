package sync

import (
	"encoding/json"
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/config"
	"github.com/go-co-op/gocron"
)

// ListConfiguredRegistrySyncers returns the contents of the config.ConfiguredRegistries as formatted JSON
func ListConfiguredRegistrySyncers() (string, error) {
	j, e := json.MarshalIndent(config.ConfiguredRegistries, "", " ")
	if e != nil {
		return "", e
	}
	return string(j), nil
}

// ListActiveRegistrySyncers returns the contents of the global sync.SyncerPool as formatted JSON
func ListActiveRegistrySyncers(syncers map[string]*Syncer) (string, error) {
	j, e := json.MarshalIndent(syncers, "", " ")
	if e != nil {
		return "", e
	}
	return string(j), nil
}

// PauseRegistry will 'pause' a registry by removing the gocron task from the pool, preventing
// further executions. It is the callers responsibility to remove the Syncer from the SyncerPool.
func PauseRegistry(syncer *Syncer, scheduler *gocron.Scheduler) error {
	syncer.EndContext()
	err := scheduler.RemoveByTag(syncer.Tag())
	if err != nil {
		return err
	}
	return nil
}

// ResumeRegistry will create a new Syncer from the config.ConfiguredRegistries list and create
// a new gocron.Job using the configured schedule.
func ResumeRegistry(syncName string, pool *SyncerPool) (*Syncer, *gocron.Job, error) {
	registry, err := config.ConfiguredRegistries.GetRegistry(syncName)
	if err != nil {
		return nil, nil, pkg.RegistryNotFound
	}

	syncer, job, err := SetupRegistryJob(registry, pool.CronJobScheduler)
	if err != nil {
		return nil, nil, err
	}

	return syncer, job, nil
}
