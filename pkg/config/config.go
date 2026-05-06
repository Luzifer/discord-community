// Package config loads and validates the bot configuration file.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	korvike "github.com/Luzifer/korvike/functions"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"

	"github.com/Luzifer/discord-community/pkg/attributestore"
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
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing config")
		}
	}()

	configContent, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	tpl, err := template.New("config").Funcs(korvike.GetFunctionMap()).Parse(string(configContent))
	if err != nil {
		return nil, fmt.Errorf("parsing config file template: %w", err)
	}

	renderedConfig := new(bytes.Buffer)
	if err = tpl.Execute(renderedConfig, nil); err != nil {
		return nil, fmt.Errorf("rendering config template: %w", err)
	}

	var (
		decoder = yaml.NewDecoder(renderedConfig)
		tmp     File
	)

	decoder.KnownFields(true)
	if err = decoder.Decode(&tmp); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	return &tmp, nil
}
