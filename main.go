package main

import (
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/config"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/load"
	"github.com/HarrisonWAffel/playground/picture-book/pkg/sync"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"time"
)

func main() {
	app := &cli.App{
		Name:        "picture-book",
		HelpName:    "",
		Usage:       "",
		UsageText:   "",
		ArgsUsage:   "",
		Version:     "",
		Description: "a utility for loading image registries",
		Commands: []*cli.Command{
			{
				Name:        "sync",
				Description: "start an automated process to continuously sync a registries with images",
				Aliases:     []string{"s"},
				Action:      sync.BeginSynchronization,
			},
			{
				Name:    "load",
				Aliases: []string{"ld"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "registry",
						Value:    "",
						Required: false,
						Usage:    "the hostname of the registry you want to load",
					},
					&cli.BoolFlag{
						Name:     "all",
						Value:    false,
						Required: false,
					},
				},
				Description: "load a set of images into a repository using the script configured in config.yaml",
				Action:      load.Load,
			}},
		Flags:                     []cli.Flag{},
		EnableBashCompletion:      false,
		HideHelp:                  false,
		HideHelpCommand:           false,
		HideVersion:               false,
		BashComplete:              nil,
		Before:                    nil,
		After:                     nil,
		Action:                    nil,
		CommandNotFound:           nil,
		OnUsageError:              nil,
		InvalidFlagAccessHandler:  nil,
		Compiled:                  time.Time{},
		Authors:                   nil,
		Copyright:                 "",
		Reader:                    nil,
		Writer:                    nil,
		ErrWriter:                 nil,
		ExitErrHandler:            nil,
		Metadata:                  nil,
		ExtraInfo:                 nil,
		CustomAppHelpTemplate:     "",
		SliceFlagSeparator:        "",
		DisableSliceFlagSeparator: false,
		UseShortOptionHandling:    false,
		Suggest:                   false,
		AllowExtFlags:             false,
		SkipFlagParsing:           false,
	}
	pkg.Logger = &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.InfoLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: pkg.TimeFormat,
			LogFormat:       "[%time%] %msg%\n",
		},
	}

	pkg.ErrLogger = &logrus.Logger{
		Out:   os.Stderr,
		Level: logrus.InfoLevel,
		Formatter: &easy.Formatter{
			TimestampFormat: pkg.TimeFormat,
			LogFormat:       "[%lvl%][%time%] %msg%\n",
		},
	}
	config.Setup()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
