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
		BotToken       string `flag:"bot-token" description:"Token from the App Bot User section"`
		GuildID        string `flag:"guild-id" description:"ID of the Discord server (guild)"`
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	crontab = cron.New()
	discord *discordgo.Session

	discordHandlers []interface{}

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("tezrian-discord %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	var err error

	// Connect to Discord
	if discord, err = discordgo.New(strings.Join([]string{"Bot", cfg.BotToken}, " ")); err != nil {
		log.WithError(err).Fatal("Unable to create discord client")
	}

	discord.Identify.Intents = discordgo.IntentsAll

	for _, hdl := range discordHandlers {
		discord.AddHandler(hdl)
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
