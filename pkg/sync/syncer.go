package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

// Syncer handles image synchronization for docker registries.
type Syncer struct {
	context.Context    `json:"-"`
	context.CancelFunc `json:"-"`
	pkg.SyncerBase
	client *client.Client
}

func BuildRegistrySyncer(ctx context.Context, cancel context.CancelFunc, registry pkg.Registry) (*Syncer, string, error) {
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
			JobTag:            tag,
			RemoveLocalImages: registry.DeleteLocalImages,
			RegistryHostName:  registry.Hostname,
			Repository:        registry.Repository,
			PullAuth:          registry.PullAuthConfig,
			PushAuth:          registry.PushAuthConfig,
			Executor: pkg.Executor{
				File: registry.SyncerScript,
				Args: registry.SyncerScriptArgs,
			},
		},
		client: dockerClient,
	}

	return &syncer, tag, nil
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

func (d *Syncer) RemoveImage(image, retagged string) error {
	// get the image so we have its ID
	switch viper.GetString("display") {
	case "spinner":
		return RemoveWithSpinner(d.Context, d.client, image, retagged)
	default:
		return RemoveWithoutSpinner(d.Context, d.client, image, retagged)
	}
}

func (d *Syncer) Retag(ctx context.Context, image string) (string, error) {
	switch viper.GetString("display") {
	case "spinner":
		return RetagWithSpinner(ctx, d.client, image, d.RegistryHostName, d.Repository)
	default:
		return RetagWithoutSpinner(ctx, d.client, image, d.RegistryHostName, d.Repository)
	}
}

func (d *Syncer) ExecScript() ([]string, error) {
	return d.Executor.ExecScript()
}

func (d *Syncer) ChangePeriod(cron string) {

	// todo; still debating this function
	//  and if we should allow the API to
	//  change registry configurations outside
	//  of the config.yaml. The changes won't persist
	//  so I don't see much point
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

type Status struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}
