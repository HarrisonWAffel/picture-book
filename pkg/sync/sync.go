package sync

import (
	"context"
	"fmt"
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/config"

	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"time"
)

func BeginSynchronization(ctx *cli.Context) error {
	pkg.Logger.Infof("Setting up registry synchronizers")
	cronRunner := gocron.NewScheduler(time.UTC)
	pool := &SyncerPool{
		Syncers:          make(map[string]*Syncer),
		CronJobScheduler: cronRunner,
		Context:          ctx.Context,
	}

	for _, registry := range config.ConfiguredRegistries {
		if _, ok := pool.Syncers[registry.Hostname]; ok {
			return fmt.Errorf("fatal: dupliacte registry found (%s), each registry hostname may only be configured once", registry.Hostname)
		}

		syncer, _, err := SetupRegistryJob(registry, cronRunner)
		if err != nil {
			pkg.ErrLogger.Errorf("error encountered setting up registry sync: %v", err)
			continue
		}

		pool.Syncers[registry.Hostname] = syncer
	}

	if viper.GetBool("api.enabled") {
		pkg.Logger.Infof("Starting Sync server...")
		go StartServer(pool)
		pkg.Logger.Infof("Sync server up!")
		pkg.Logger.Infof("Access the Syncrhonization Server on http://localhost:%s", viper.GetString("api.port"))
	}

	pkg.Logger.Infof("Registry synchronizers created!")

	// Run all configured jobs forever!
	pool.CronJobScheduler.StartBlocking()

	return nil
}

func SetupRegistryJob(registry pkg.Registry, cronRunner *gocron.Scheduler) (*Syncer, *gocron.Job, error) {
	ctx, cancel := context.WithCancel(context.Background())
	syncer, tag, err := BuildRegistrySyncer(ctx, cancel, registry)
	if err != nil {
		return nil, nil, err
	}

	job, err := cronRunner.Cron(registry.SyncPeriod).Do(syncer.Process)
	if err != nil {
		return nil, nil, err
	}

	// tag the job so the sever can query it
	job.Tag(tag)
	// If a job has not finished we should not re-run it.
	job.SingletonMode()
	syncer.Job = job
	return syncer, job, nil
}
