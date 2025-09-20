package config

import (
	"bytes"
	"io"
	"os"
	"text/template"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	korvike "github.com/Luzifer/korvike/functions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"
)

type (
	// File represents the contents of a config file
	File struct {
		BotToken      string `yaml:"bot_token"`
		GuildID       string `yaml:"guild_id"`
		StoreLocation string `yaml:"store_location"`

		ModuleConfigs []ModuleConfig `yaml:"module_configs"`
	}

	// ModuleConfig contains the configuration for a module
	ModuleConfig struct {
		ID         string                              `yaml:"id"`
		Type       string                              `yaml:"type"`
		Attributes attributestore.ModuleAttributeStore `yaml:"attributes"`
	}
)

// NewFromFile reads the configuration from the given file
func NewFromFile(filename string) (*File, error) {
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
		tmp     File
	)

	decoder.KnownFields(true)
	return &tmp, errors.Wrap(decoder.Decode(&tmp), "decoding config")
}
