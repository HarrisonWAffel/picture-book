package pkg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

var TimeFormat = time.RFC850
var Logger *logrus.Logger

type Registry struct {
	// Hostname of the registry, not including https
	Hostname string `yaml:"hostname"`
	// Repository is a prefix added to an image which denotes a particular base repository
	// for a collection of images.
	Repository string
	// PushAuthConfig is the dockerconfigjson value for the registry that will be pushed to
	PushAuthConfig string `yaml:"pushAuthConfig"`
	// PullAuthConfig is the dockerconfigjson value for the registry that will be pulled from
	PullAuthConfig string `yaml:"pullAuthConfig"`
	// SyncPeriod is a cron configuration
	SyncPeriod string `yaml:"syncPeriod"`
	// SyncerScript points to the script that should be
	// used to get new images for syncing
	SyncerScript string `yaml:"syncerScript"`
	// SyncerScriptArgs is a string containing flags and arguments which can be passed to as syncer script
	SyncerScriptArgs string `yaml:"syncerScriptArgs"`
	// RegistryProvider is the type of registry (docker / harbor)
	RegistryProvider string `yaml:"registryProvider"`
}

type Registries []Registry

var RegistryNotFound = errors.New("could not find provided registry by hostname")

func (r Registries) GetRegistry(hostname string) (Registry, error) {
	for _, registry := range r {
		if registry.Hostname == hostname {
			return registry, RegistryNotFound
		}
	}
	return Registry{}, RegistryNotFound
}

type Executor struct {
	Args string
	File string
}

type Details struct {
	NumberOfSyncs int
	Created       time.Time
}

type SyncerBase struct {
	context.Context    `json:"-"`
	context.CancelFunc `json:"-"`
	Details            Details
	Executor           Executor `json:"-"`
	RegistryHostName   string
	Repository         string
	JobTag             string
	PullAuth           string `json:"-"`
	PushAuth           string `json:"-"`
	Job                *gocron.Job
}

func (e *Executor) ExecScript() ([]string, error) {
	// todo; this should also accept args
	//  and the error handling needs to be better
	Logger.Infof("Executing %s", "./sync-scripts/"+e.File)

	cmd := exec.Command("./sync-scripts/" + e.File)
	cmd.Args = strings.Split(e.Args, " ")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return strings.Split(string(output), "\n"), nil
}

func ReTag(image, host, repository string) string {
	split := strings.Split(image, "/")
	newHost := host
	if repository != "" {
		newHost = newHost + "/" + repository
	}
	newImg := strings.ReplaceAll(image, split[0], newHost)
	return newImg
}

func BuildPullOptions(auth, hostname string) types.ImagePullOptions {
	ops := types.ImagePullOptions{
		RegistryAuth: auth,
	}
	if ops.RegistryAuth == "" {
		ops.RegistryAuth = BuildEncodedAuthConfig(types.AuthConfig{
			ServerAddress: hostname,
		})
	}
	return ops
}

func BuildPushOptions(auth, hostname string) types.ImagePushOptions {
	ops := types.ImagePushOptions{
		RegistryAuth: auth,
	}
	if ops.RegistryAuth == "" {
		ops.RegistryAuth = BuildEncodedAuthConfig(types.AuthConfig{
			ServerAddress: hostname,
		})
	}
	return ops
}

func BuildEncodedAuthConfig(config types.AuthConfig) string {
	authConfigBytes, _ := json.Marshal(config)
	return base64.URLEncoding.EncodeToString(authConfigBytes)
}

func BuildCronJobTag(registryHostName string) string {
	return registryHostName + "-job"
}
