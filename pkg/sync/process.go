package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"strings"
	"time"

	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/docker/docker/client"
	"github.com/theckman/yacspin"
)

func (d *Syncer) Process() {
	images, err := d.ExecScript()
	if err != nil {
		pkg.ErrLogger.Errorf("error encountered when executing syncer script %s. Exiting script execution and image syncing process: %v", d.Executor.File, err)
		return
	}
	pkg.Logger.Infof("Beginning synchronization for %s", d.RegistryHostName)

SyncLoop:
	for _, image := range images {
		if image == "" {
			continue
		}

		// check for any cancel signals from the API
		select {
		case <-d.Context.Done():
			pkg.Logger.Infof("Canceling image synchronization due to syncer pause via API")
			break SyncLoop
		default:

		}

		// check if the target registry already has the image and tag being processed
		alreadyPushed, err := d.ImageExistsOnRegistry(image)
		if err != nil {
			pkg.ErrLogger.Errorf("Error encountered while checking if image has already been pushed to %s: %v", d.RegistryHostName, err)
			continue
		}

		if alreadyPushed {
			pkg.Logger.Infof("%s has already been retagged and pushed to remote repository!", image)
			// nothing to do!
			continue
		}

		err = d.Pull(image)
		if err != nil && !errors.Is(err, context.Canceled) {
			pkg.ErrLogger.Errorf("Error encountered while pulling %s: %v", image, err)
			continue
		}

		reTaggedImage, err := d.Retag(d.Context, image)
		if err != nil && !errors.Is(err, context.Canceled) {
			pkg.ErrLogger.Errorf("Could not retag image '%s' -> '%s': %v", image, reTaggedImage, err)
			continue
		}

		err = d.Push(reTaggedImage)
		if err != nil && !errors.Is(err, context.Canceled) {
			pkg.ErrLogger.Errorf("Error encountered while pushing %s to %s: %v", reTaggedImage, d.RegistryHostName, err)
			continue
		}

		if d.RemoveLocalImages {
			// this also removes the retagged image since they share the same image ID
			err = d.RemoveImage(image, reTaggedImage)
			if err != nil {
				pkg.ErrLogger.Errorf("couldn't delete locally held image %s: %v", image, err)
			}
		}
	}
	pkg.Logger.Infof("Done synchronizing images for %s", d.RegistryHostName)
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

func RemoveImage(ctx context.Context, client *client.Client, image string) error {
	img, _, err := client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return err
	}
	// force delete the image using its ID
	_, err = client.ImageRemove(ctx, img.ID, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	return err
}

func Retag(ctx context.Context, client *client.Client, image, reTaggedImage string) (string, error) {
	err := client.ImageTag(ctx, image, reTaggedImage)
	if err != nil {
		return "", err
	}
	return reTaggedImage, nil
}

// push / pull logic

func PushWithDisplayFunc(ctx context.Context, client *client.Client, reTaggedImage, hostname, auth string, f func(scanner *bufio.Scanner, op, image, hostname string)) error {
	r, err := client.ImagePush(ctx, reTaggedImage, pkg.BuildPushOptions(auth, hostname))
	if err != nil {
		return err
	}
	defer r.Close()
	f(bufio.NewScanner(r), "Pushing", reTaggedImage, hostname)
	return nil
}

func PullWithDisplayFunc(ctx context.Context, client *client.Client, image, hostname, auth string, f func(scanner *bufio.Scanner, op, image, hostname string)) error {
	r, err := client.ImagePull(ctx, image, pkg.BuildPullOptions(auth, hostname))
	if err != nil {
		if strings.Contains(err.Error(), "repository does not exist") {
			return pkg.ImageNotFound
		}
		return err
	}
	defer r.Close()
	f(bufio.NewScanner(r), "Pulling", image, hostname)
	return nil
}

// push / pull display logic

func PushPullSpinner(scanner *bufio.Scanner, op, image, _ string) {
	spinner, _ := yacspin.New(cfg)
	spinner.Suffix(fmt.Sprintf("%s %s %s", op, time.Now().Format(pkg.TimeFormat), image))
	spinner.Start()
	defer spinner.Stop()

	var status DockerStatusOutput
	for {
		if !scanner.Scan() {
			spinner.Suffix(fmt.Sprintf("[%s] Done pushing %s", time.Now().Format(pkg.TimeFormat), image))
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		spinner.Message(fmt.Sprintf("%s %s", status.Status, status.Progress))
	}
}

func PushPullStdDisplay(scanner *bufio.Scanner, _, image, hostname string) {
	var status DockerStatusOutput
	for {
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &status)
		if status.Progress == "" {
			fmt.Println(fmt.Sprintf("%s: %s:", hostname, status.Status))
		} else {
			fmt.Println(fmt.Printf("%s: %s: %s", image, status.Status, status.Progress))
		}
	}
}
