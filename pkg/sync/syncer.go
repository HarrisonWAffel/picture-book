package sync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/theckman/yacspin"
	"io"
	"net/http"
	"strings"
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

// ImageExistsOnRegistry checks if the given image already exists on the host being pushed to.
// if the image tag is 'latest', or is empty, an image will always be processed.
func (d *Syncer) ImageExistsOnRegistry(image string) (bool, error) {
	if image == "" {
		return false, fmt.Errorf("encountered an empty image name")
	}

	// we retag first to append the specified repository
	// we then strip the hostname since it will be specified in the URL built later
	// we then separate the image name and tag to use later
	imgWithoutTag, tag := pkg.GetImageAndTag(pkg.ImageWithoutHost(pkg.ReTag(image, d.RegistryHostName, d.Repository), d.RegistryHostName))

	// latest tags will have their manifests updated
	// regularly, so we should always try to pull and push
	// the most recent version. If we aren't given a tag
	// we should treat it as latest
	if tag == "latest" || tag == "" {
		return false, nil
	}

	// making an HTTP request to the registry being pushed to is easier than
	// creating a whole new docker client for this.
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/v2/%s/tags/list", d.RegistryHostName, imgWithoutTag), nil)
	if err != nil {
		return false, err
	}

	// setup basic auth
	if d.PushAuth != "" {
		auths := strings.Split(d.PushAuth, ":")
		if len(auths) != 2 {
			return false, fmt.Errorf("pushConfig for %s is improperly formatted, expected format is 'username:password'", d.RegistryHostName)
		}
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", auths[0], auths[1]))))
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error encountered making HTTP request to %s: %v", d.RegistryHostName, err)
	}

	// don't bother reading the response body if we get a 404
	if r.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if r.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("pushAuthConfig for %s is invalid", d.RegistryHostName)
	}

	var RegistryResponse struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	defer r.Body.Close()
	out, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("error encountered reading HTTP response from %s: %v", d.RegistryHostName, err)
	}

	// since the json response body varies in format we want to
	// use a map to check if we got any errors before we marshal
	// the response body into a more useful struct
	unstructuredResponse := make(map[string]interface{})
	err = json.NewDecoder(bytes.NewReader(out)).Decode(&unstructuredResponse)
	if err != nil {
		return false, fmt.Errorf("error encountered reading HTTP response from %s: %v", d.RegistryHostName, err)
	}

	if _, ok := unstructuredResponse["errors"]; ok {
		return false, fmt.Errorf("encountered an error handling unstructured response body: unstructured response = %v", unstructuredResponse)
	}

	// if we got a valid response marshal it into a struct to avoid casting
	err = json.NewDecoder(bytes.NewReader(out)).Decode(&RegistryResponse)
	if err != nil {
		return false, err
	}

	for _, foundTag := range RegistryResponse.Tags {
		if foundTag == tag {
			return true, nil
		}
	}

	return false, nil
}

func (d *Syncer) Pull(image string) error {
	switch viper.GetString("display") {
	case "spinner":
		return PullWithDisplayFunc(d.Context, d.client, image, d.RegistryHostName, d.PullAuth, PushPullSpinner)
	default:
		return PullWithDisplayFunc(d.Context, d.client, image, d.RegistryHostName, d.PullAuth, PushPullStdDisplay)
	}
}

func (d *Syncer) Push(image string) error {
	switch viper.GetString("display") {
	case "spinner":
		return PushWithDisplayFunc(d.Context, d.client, image, d.RegistryHostName, d.PullAuth, PushPullSpinner)
	default:
		return PushWithDisplayFunc(d.Context, d.client, image, d.RegistryHostName, d.PullAuth, PushPullStdDisplay)
	}
}

func (d *Syncer) RemoveImage(image, retagged string) error {
	// get the image so we have its ID
	switch viper.GetString("display") {
	case "spinner":
		spinner, _ := yacspin.New(cfg)
		spinner.Suffix(fmt.Sprintf("[%s] Removing locally held images %s, %s", time.Now().Format(pkg.TimeFormat), image, retagged))
		spinner.Start()
		defer spinner.Stop()
		return RemoveImage(d.Context, d.client, image)
	default:
		pkg.Logger.Infof("Removing locally held images %s, %s", image, retagged)
		return RemoveImage(d.Context, d.client, image)
	}
}

func (d *Syncer) Retag(ctx context.Context, image string) (string, error) {
	reTaggedImage := pkg.ReTag(image, d.RegistryHostName, d.Repository)
	switch viper.GetString("display") {
	case "spinner":
		spinner, _ := yacspin.New(cfg)
		spinner.Suffix(fmt.Sprintf("[%s] Retagging %s -> %s: ", time.Now().Format(pkg.TimeFormat), image, reTaggedImage))
		spinner.Start()
		defer spinner.Stop()
		return Retag(ctx, d.client, image, reTaggedImage)
	default:
		pkg.Logger.Infof("Retagging %s -> %s", image, reTaggedImage)
		return Retag(ctx, d.client, image, reTaggedImage)
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

type DockerStatusOutput struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}
