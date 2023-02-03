package pkg

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
var ErrLogger *logrus.Logger

type Registry struct {
	// Hostname of the registry, not including https
	Hostname string `yaml:"hostname"`
	// Repository is a prefix added to an image which denotes a particular base repository
	// for a collection of images.
	Repository string `yaml:"repository"`
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
	// DeleteLocalImages instructs the syncer to remove locally downloaded images after pushing them to a registry.
	DeleteLocalImages bool `yaml:"deleteLocalImages"`
}

type Registries []Registry

var RegistryNotFound = errors.New("could not find provided registry by hostname")
var ImageNotFound = errors.New("repository does not exist")

func (r Registries) GetRegistry(hostname string) (Registry, error) {
	for _, registry := range r {
		if registry.Hostname == hostname {
			return registry, nil
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
	RemoveLocalImages  bool
	PullAuth           string `json:"-"`
	PushAuth           string `json:"-"`
	Job                *gocron.Job
}

func (e *Executor) ExecScript() ([]string, error) {

	Logger.Infof("Executing %s", e.File)

	cmd := exec.Command(e.File, strings.Split(e.Args, " ")...)
	fmt.Println(cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return strings.Split(string(output), "\n"), nil
}

func ReTag(image, host, repository string) string {
	var newImg string
	if strings.HasPrefix(image, repository) {
		newImg = host + "/" + image
	} else {
		if repository != "" {
			newImg = host + "/" + repository + "/" + image
		} else {
			newImg = host + "/" + image
		}
	}
	return newImg
}

func ImageWithoutHost(image, host string) string {
	return strings.ReplaceAll(image, host+"/", "")
}

func GetImageAndTag(image string) (string, string) {
	imgParts := strings.Split(image, ":")
	if len(imgParts) == 2 {
		return imgParts[0], imgParts[1]
	} else {
		return imgParts[0], ""
	}
}

func BuildPullOptions(auth, hostname string) types.ImagePullOptions {
	ops := types.ImagePullOptions{}
	username := ""
	password := ""
	auths := strings.Split(auth, ":")
	if auth != "" && len(auths) == 2 {
		username = auths[0]
		password = auths[1]
	}
	if ops.RegistryAuth == "" {
		ops.RegistryAuth = BuildEncodedAuthConfig(types.AuthConfig{
			Username:      username,
			Password:      password,
			ServerAddress: hostname,
		})
	}
	return ops
}

func BuildPushOptions(auth, hostname string) types.ImagePushOptions {
	ops := types.ImagePushOptions{}
	username := ""
	password := ""
	auths := strings.Split(auth, ":")
	if auth != "" && len(auths) == 2 {
		username = auths[0]
		password = auths[1]
	}
	if ops.RegistryAuth == "" {
		ops.RegistryAuth = BuildEncodedAuthConfig(types.AuthConfig{
			Username:      username,
			Password:      password,
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
