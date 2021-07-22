package main

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type (
	configFile struct {
		BotToken string `yaml:"bot_token"`
		GuildID  string `yaml:"guild_id"`

		ModuleConfigs []moduleConfig `yaml:"module_configs"`
	}

	moduleConfig struct {
		Type       string               `yaml:"type"`
		Attributes moduleAttributeStore `yaml:"attributes"`
	}
)

func newConfigFromFile(filename string) (*configFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	defer f.Close()

	var (
		decoder = yaml.NewDecoder(f)
		tmp     configFile
	)

	decoder.SetStrict(true)
	return &tmp, errors.Wrap(decoder.Decode(&tmp), "decoding config")
}
