package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	httpHelpers "github.com/Luzifer/go_helpers/v2/http"
	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Config         string `flag:"config,c" default:"config.yaml" description:"Path to config file"`
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	config *configFile

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("discord-community %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	var (
		crontab = cron.New()
		discord *discordgo.Session
		err     error
	)

	if config, err = newConfigFromFile(cfg.Config); err != nil {
		log.WithError(err).Fatal("Unable to load config file")
	}

	// Connect to Discord
	if discord, err = discordgo.New(strings.Join([]string{"Bot", config.BotToken}, " ")); err != nil {
		log.WithError(err).Fatal("Unable to create discord client")
	}

	discord.Identify.Intents = discordgo.IntentsAll

	var enabledModules int
	for i, mc := range config.ModuleConfigs {
		logger := log.WithFields(log.Fields{
			"idx":    i,
			"module": mc.Type,
		})
		mod := GetModuleByName(mc.Type)
		if mod == nil {
			logger.Fatal("Found configuration for unsupported module")
		}

		if err = mod.Initialize(crontab, discord, mc.Attributes); err != nil {
			logger.WithError(err).Fatal("Unable to initialize module")
		}

		enabledModules++
		logger.Debug("Enabled module")
	}

	if enabledModules == 0 {
		log.Warn("No modules were enabled, quitting now")
		return
	}

	if err = discord.Open(); err != nil {
		log.WithError(err).Fatal("Unable to connect discord client")
	}
	defer discord.Close()
	log.Debug("Discord connected")

	// Run Crontab
	crontab.Start()
	defer crontab.Stop()
	log.Debug("Crontab started")

	// Run HTTP server
	var h http.Handler = http.DefaultServeMux
	h = httpHelpers.GzipHandler(h)
	h = httpHelpers.NewHTTPLogHandler(h)

	http.ListenAndServe(cfg.Listen, h)
}
