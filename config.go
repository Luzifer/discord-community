package main

import (
	"bytes"
	"io"
	"os"
	"text/template"

	korvike "github.com/Luzifer/korvike/functions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"
)

type (
	configFile struct {
		BotToken      string `yaml:"bot_token"`
		GuildID       string `yaml:"guild_id"`
		StoreLocation string `yaml:"store_location"`

		ModuleConfigs []moduleConfig `yaml:"module_configs"`
	}

	moduleConfig struct {
		ID         string               `yaml:"id"`
		Type       string               `yaml:"type"`
		Attributes moduleAttributeStore `yaml:"attributes"`
	}
)

func newConfigFromFile(filename string) (*configFile, error) {
	f, err := os.Open(filename) //#nosec:G304 // Intended to load specified config
	if err != nil {
		return nil, errors.Wrap(err, "opening config file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing config")
		}
	}()

	configContent, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "reading config file")
	}

	tpl, err := template.New("config").Funcs(korvike.GetFunctionMap()).Parse(string(configContent))
	if err != nil {
		return nil, errors.Wrap(err, "parsing config file template")
	}

	renderedConfig := new(bytes.Buffer)
	if err = tpl.Execute(renderedConfig, nil); err != nil {
		return nil, errors.Wrap(err, "rendering config template")
	}

	var (
		decoder = yaml.NewDecoder(renderedConfig)
		tmp     configFile
	)

	decoder.KnownFields(true)
	return &tmp, errors.Wrap(decoder.Decode(&tmp), "decoding config")
}
