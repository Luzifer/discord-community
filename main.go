package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/discord-community/pkg/config"
	"github.com/Luzifer/discord-community/pkg/modules"
	httpHelpers "github.com/Luzifer/go_helpers/v2/http"
	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Config         string `flag:"config,c" default:"config.yaml" description:"Path to config file"`
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	confFile *config.File
	store    *modules.MetaStore

	version = "dev"
)

func initApp() (err error) {
	rconfig.AutoEnv(true)
	if err = rconfig.ParseAndValidate(&cfg); err != nil {
		return fmt.Errorf("parsing CLI options: %w", err)
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parsing log-level: %w", err)
	}
	logrus.SetLevel(l)

	return nil
}

//nolint:funlen,gocyclo
func main() {
	var (
		crontab       = cron.New()
		discord       *discordgo.Session
		err           error
		activeModules []modules.Module
	)

	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		fmt.Printf("discord-community %s\n", version) //nolint:forbidigo
		os.Exit(0)
	}

	if confFile, err = config.NewFromFile(cfg.Config); err != nil {
		logrus.WithError(err).Fatal("loading config file")
	}

	if confFile.StoreLocation == "" {
		logrus.Fatal("config contains no store location")
	}

	if store, err = modules.NewMetaStoreFromDisk(confFile.StoreLocation); err != nil {
		logrus.WithError(err).Fatal("loading store")
	}

	// Connect to Discord
	if discord, err = discordgo.New(strings.Join([]string{"Bot", confFile.BotToken}, " ")); err != nil {
		logrus.WithError(err).Fatal("creating discord client")
	}

	discord.Identify.Intents = discordgo.IntentsAll

	var activeIDs []string
	for i, mc := range confFile.ModuleConfigs {
		logger := logrus.WithFields(logrus.Fields{
			"id":     mc.ID,
			"idx":    i,
			"module": mc.Type,
		})

		if str.StringInSlice(mc.ID, activeIDs) {
			logger.Error("found duplicate module ID, module will be disabled")
			continue
		}

		if mc.ID == "" {
			logger.Error("module contains no ID and will be disabled")
			continue
		}

		mod := modules.GetModuleByName(mc.Type)
		if mod == nil {
			logger.Fatal("found configuration for unsupported module")
		}

		if err = mod.Initialize(modules.ModuleInitArgs{
			ID:    mc.ID,
			Attrs: mc.Attributes,

			Crontab: crontab,
			Discord: discord,
			Config:  confFile,
			Store:   store,
		}); err != nil {
			logger.WithError(err).Fatal("initializing module")
		}

		activeModules = append(activeModules, mod)
		activeIDs = append(activeIDs, mc.ID)

		logger.Debug("enabled module")
	}

	if len(activeModules) == 0 {
		logrus.Warn("no modules were enabled, quitting now")
		return
	}

	if err = discord.Open(); err != nil {
		logrus.WithError(err).Fatal("connecting discord client")
	}
	defer discord.Close() //nolint:errcheck // Will be closed by program exit
	logrus.Debug("discord connected")

	guild, err := discord.Guild(confFile.GuildID)
	if err != nil {
		logrus.WithError(err).Fatal("getting guild for given guild-id in config: is the bot added and the ID correct?")
	}
	logrus.WithField("name", guild.Name).Info("found specified guild for operation")

	// Run Crontab
	crontab.Start()
	defer crontab.Stop()
	logrus.Debug("crontab started")

	// Execute Setup methods now after we're connected
	for i, mod := range activeModules {
		if err = mod.Setup(); err != nil {
			logrus.WithError(err).WithField("idx", i).Fatal("running setup for module")
		}
	}

	// Run HTTP server
	var h http.Handler = http.DefaultServeMux
	h = httpHelpers.GzipHandler(h)
	h = httpHelpers.NewHTTPLogHandler(h)

	server := &http.Server{
		Addr:              cfg.Listen,
		Handler:           h,
		ReadHeaderTimeout: time.Second,
	}

	logrus.WithField("version", version).Info("bot setup done, bot is now running")

	if err = server.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("listening for HTTP traffic")
	}
}
