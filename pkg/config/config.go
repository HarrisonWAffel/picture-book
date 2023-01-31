package config

import (
	"fmt"
	"github.com/HarrisonWAffel/playground/picture-book/pkg"
	"github.com/spf13/viper"
)

var ConfiguredRegistries pkg.Registries

func Setup() {
	viper.SetConfigName("")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../..")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	if err := viper.UnmarshalKey("registries", &ConfiguredRegistries); err != nil {
		panic(fmt.Errorf("Could not unmarshal config.yaml file: %w", err))
	}
}
