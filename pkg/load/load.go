package load

import (
	"context"
	"fmt"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/config"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/sync"
	"github.com/urfave/cli/v2"
)

func Load(cliCtx *cli.Context) error {
	hostname := cliCtx.String("registry")
	if hostname == "" {
		return fmt.Errorf("you must supply a registry hostname")
	}

	registry, err := config.ConfiguredRegistries.GetRegistry(hostname)
	if err != nil {
		return fmt.Errorf("could not find provided registry %s", hostname)
	}

	ctx, cancel := context.WithCancel(context.Background())
	syncer, _, err := sync.BuildDockerSyncer(ctx, cancel, registry)
	if err != nil {
		return err
	}
	syncer.Load()

	fmt.Println("Done!")
	return nil
}
