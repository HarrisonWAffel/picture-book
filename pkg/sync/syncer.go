package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Syncer handles image synchronization for docker registries.
type Syncer struct {
	context.Context    `json:"-"`
	context.CancelFunc `json:"-"`
	pkg.SyncerBase
	client *client.Client
}

func (d *Syncer) EndContext() {
	d.CancelFunc()
}

func (d *Syncer) SetContext(ctx context.Context, cancel context.CancelFunc) {
	d.Context = ctx
	d.CancelFunc = cancel
}

func (d *Syncer) GetContext() (context.Context, context.CancelFunc) {
	return d.Context, d.CancelFunc
}
func (d *Syncer) Pull(image string) error {
	switch viper.GetString("display") {
	case "spinner":
		return PullWithSpinner(d.Context, d.client, image, d.RegistryHostName, d.PullAuth)
	default:
		return PullWithoutSpinner(d.Context, d.client, image, d.RegistryHostName, d.PullAuth)
	}
}

func (d *Syncer) Push(image string) error {
	switch viper.GetString("display") {
	case "spinner":
		return PushWithSpinner(d.Context, d.client, image, d.RegistryHostName, d.PushAuth)
	default:
		return PushWithoutSpinner(d.Context, d.client, image, d.RegistryHostName, d.PushAuth)
	}
}

func (d *Syncer) Retag(ctx context.Context, image string) (string, error) {
	reTaggedImage := pkg.ReTag(image, d.RegistryHostName, d.Repository)
	err := d.client.ImageTag(ctx, image, reTaggedImage)
	if err != nil {
		return "", err
	}
	return reTaggedImage, nil
}

func (d *Syncer) ExecScript() ([]string, error) {
	return d.Executor.ExecScript()
}

func (d *Syncer) Load() {
	images, err := d.ExecScript()
	if err != nil {
		pkg.Logger.Errorf("error encountered when executing syncer script %s. Exiting script execution and image syncing process: %v", d.Executor.File, err)
		return
	}
	for _, i := range images {
		if i == "" {
			continue
		}

		if err := d.Pull(i); err != nil {
			logrus.Error(fmt.Errorf("could not pull %s, skipping", i))
			continue
		}

		reTaggedImage, err := d.Retag(d.Context, i)
		if err != nil {
			logrus.Error(fmt.Errorf("failed to retag image %s, skipping", i))
			continue
		}

		if err = d.Push(reTaggedImage); err != nil {
			// todo; should we clean it automatically?
			logrus.Error(fmt.Errorf("could not push %s, skipping. Ensure unpushed images are cleaned from host system", reTaggedImage))
			continue
		}
	}
}

func (d *Syncer) ChangePeriod(cron string) {
	// todo
	j := d.Job
	_ = j
	return
}

func (d *Syncer) Tag() string {
	return d.JobTag
}

func (d *Syncer) Info() []byte {
	d.Details.NumberOfSyncs = d.Job.RunCount()
	j, _ := json.MarshalIndent(d, "", " ")
	return j
}

func BuildDockerSyncer(ctx context.Context, cancel context.CancelFunc, registry pkg.Registry) (*Syncer, string, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &Syncer{}, "", fmt.Errorf("could not create docker client for registry %s: %w", registry.Hostname, err)
	}
	tag := pkg.BuildCronJobTag(registry.Hostname)
	syncer := Syncer{
		Context:    ctx,
		CancelFunc: cancel,
		SyncerBase: pkg.SyncerBase{
			Details: pkg.Details{
				Created: time.Now(),
			},
			JobTag:           tag,
			RegistryHostName: registry.Hostname,
			Repository:       registry.Repository,
			PullAuth:         registry.PullAuthConfig,
			PushAuth:         registry.PushAuthConfig,
			Executor: pkg.Executor{
				File: registry.SyncerScript,
				Args: registry.SyncerScriptArgs,
			},
		},
		client: dockerClient,
	}

	return &syncer, tag, nil
}

type Status struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}
