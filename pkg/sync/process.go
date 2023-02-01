package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
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

		err := d.Pull(image)
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

// implementation for syncer functions push, pull, retag, and remove.
// I recognize the implementation is duplicated for spinner vs non-spinner, but I like the spinner :)
// should de-duplicate at some point in the future (maybe).

func RemoveWithSpinner(ctx context.Context, client *client.Client, image, reTaggedImage string) error {
	spinner, _ := yacspin.New(cfg)
	spinner.Suffix(fmt.Sprintf("[%s] Removing locally held images %s, %s", time.Now().Format(pkg.TimeFormat), image, reTaggedImage))
	spinner.Start()
	defer spinner.Stop()
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

func RemoveWithoutSpinner(ctx context.Context, client *client.Client, image, reTaggedImage string) error {
	pkg.Logger.Infof("Removing locally held images %s, %s", image, reTaggedImage)
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

func RetagWithSpinner(ctx context.Context, client *client.Client, image, hostname, repository string) (string, error) {
	reTaggedImage := pkg.ReTag(image, hostname, repository)
	spinner, _ := yacspin.New(cfg)
	spinner.Suffix(fmt.Sprintf("[%s] Retagging %s -> %s: ", time.Now().Format(pkg.TimeFormat), image, reTaggedImage))
	spinner.Start()
	defer spinner.Stop()
	err := client.ImageTag(ctx, image, reTaggedImage)
	if err != nil {
		return "", err
	}
	return reTaggedImage, nil
}

func RetagWithoutSpinner(ctx context.Context, client *client.Client, image, hostname, repository string) (string, error) {
	reTaggedImage := pkg.ReTag(image, hostname, repository)
	pkg.Logger.Infof("Retagging %s -> %s", image, reTaggedImage)
	err := client.ImageTag(ctx, image, reTaggedImage)
	if err != nil {
		return "", err
	}
	return reTaggedImage, nil
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
	defer r.Close()
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
			fmt.Println(fmt.Printf("%s: %s: %s", image, status.Status, status.Progress))
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
