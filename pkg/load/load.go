package load

import (
	"context"
	"fmt"
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/config"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/sync"
	"github.com/urfave/cli/v2"
)

func Load(cliCtx *cli.Context) error {
	hostname := cliCtx.String("registry")
	all := cliCtx.Bool("all")
	if hostname == "" && !all {
		return fmt.Errorf("you must supply a registry hostname to load a single registry, or pass the --all flag to load all registries")
	}

	if all {
		return loadAll()
	} else {
		return loadSingle(hostname)
	}
}

func loadSingle(hostname string) error {
	registry, err := config.ConfiguredRegistries.GetRegistry(hostname)
	if err != nil {
		return fmt.Errorf("could not find provided registry %s", hostname)
	}

	ctx, cancel := context.WithCancel(context.Background())
	syncer, _, err := sync.BuildRegistrySyncer(ctx, cancel, registry)
	if err != nil {
		return err
	}
	syncer.Process()

	pkg.Logger.Infof("Done!")
	return nil
}

func loadAll() error {
	for _, registry := range config.ConfiguredRegistries {
		ctx, cancel := context.WithCancel(context.Background())
		syncer, _, err := sync.BuildRegistrySyncer(ctx, cancel, registry)
		if err != nil {
			return err
		}
		syncer.Process()
	}
	pkg.Logger.Infof("Done!")
	return nil
}
