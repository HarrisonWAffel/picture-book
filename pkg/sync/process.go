package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"time"

	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/theckman/yacspin"
)

func ProcessRegistry(ctx context.Context, syncer *Syncer) {
	pkg.Logger.Infof("Beginning to sync images for %s", syncer.RegistryHostName)
	images, err := syncer.Executor.ExecScript()
	if err != nil {
		pkg.Logger.Errorf("Failed to execute syncer script '%s', skipping synchronization attempt for host '%s'", syncer.Executor.File, syncer.RegistryHostName)
		return
	}

SyncLoop:
	for _, image := range images {
		if image == "" {
			continue
		}

		// check for any cancel signals
		select {
		case <-syncer.Context.Done():
			pkg.Logger.Infof("Canceling image synchronization due to syncer pause via API")
			break SyncLoop
		default:

		}

		err := syncer.Pull(image)
		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.Errorf("Error encountered while pulling %s: %v", image, err)
			continue
		}

		// retag image for registry
		reTaggedImage, err := syncer.Retag(ctx, image)
		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.Errorf("Could not retag image '%s' -> '%s': %v", image, reTaggedImage, err)
			continue
		}

		err = syncer.Push(reTaggedImage)
		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.Errorf("Error encountered while pushing %s to %s: %v", reTaggedImage, syncer.RegistryHostName, err)
			continue
		}
	}

	pkg.Logger.Infof("Done syncing images for %s", syncer.RegistryHostName)
}

var cfg = yacspin.Config{
	Frequency:       100 * time.Millisecond,
	CharSet:         yacspin.CharSets[59],
	Suffix:          "",
	SuffixAutoColon: true,
	Message:         "",
	StopCharacter:   "✓",
	StopColors:      []string{"fgGreen"},
}

func PullWithSpinner(ctx context.Context, client *client.Client, image, hostname, auth string) error {
	spinner, _ := yacspin.New(cfg)
	spinner.Suffix(fmt.Sprintf("[%s] Pulling %s: ", time.Now().Format(pkg.TimeFormat), image))
	spinner.Start()
	defer spinner.Stop()

	r, err := client.ImagePull(ctx, image, pkg.BuildPullOptions(auth, hostname))
	if err != nil {
		return err
	}

	var status Status
	scanner := bufio.NewScanner(r)
	for {
		if !scanner.Scan() {
			spinner.Suffix(fmt.Sprintf("[%s] Done pulling %s", time.Now().Format(pkg.TimeFormat), image))
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		spinner.Message(fmt.Sprintf("%s %s", status.Status, status.Progress))
	}

	return nil
}

func PushWithSpinner(ctx context.Context, client *client.Client, reTaggedImage, hostname, auth string) error {
	spinner, _ := yacspin.New(cfg)
	spinner.Suffix(fmt.Sprintf("%s Pushing %s", time.Now().Format(pkg.TimeFormat), reTaggedImage))
	spinner.Start()
	defer spinner.Stop()

	r, err := client.ImagePush(ctx, reTaggedImage, pkg.BuildPushOptions(auth, hostname))
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	var status Status
	for {
		if !scanner.Scan() {
			spinner.Suffix(fmt.Sprintf("[%s] Done pushing %s", time.Now().Format(pkg.TimeFormat), reTaggedImage))
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		spinner.Message(fmt.Sprintf("%s %s", status.Status, status.Progress))
	}
	return nil
}

func PullWithoutSpinner(ctx context.Context, client *client.Client, image, hostname, auth string) error {
	r, err := client.ImagePull(ctx, image, pkg.BuildPullOptions(auth, hostname))
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(r)
	var status Status
	for {
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		if status.Progress == "" {
			fmt.Println(fmt.Sprintf("%s: %s:", hostname, status.Status))
		} else {
			fmt.Println(fmt.Printf("%s: %s: %s %s", image, status.Status, status.Progress))
		}
	}
	return nil
}

func PushWithoutSpinner(ctx context.Context, client *client.Client, reTaggedImage, hostname, auth string) error {
	r, err := client.ImagePush(ctx, reTaggedImage, pkg.BuildPushOptions(auth, hostname))
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(r)
	var status Status
	for {
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		if status.Progress == "" {
			fmt.Println(fmt.Sprintf("%s: %s", hostname, status.Status))
		} else {
			fmt.Println(fmt.Printf("%s: %s %s", reTaggedImage, status.Status, status.Progress))
		}
	}
	return nil
}
