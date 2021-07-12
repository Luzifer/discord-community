package main

import "os"

var (
	discordAnnouncementChannel = os.Getenv("DISCORD_ANNOUNCEMENT_CHANNEL")
	discordLiveChannel         = os.Getenv("DISCORD_LIVE_CHANNEL")
	twitchChannelID            = os.Getenv("TWITCH_CHANNEL_ID")
	twitchClientID             = os.Getenv("TWITCH_CLIENT_ID")
	twitchToken                = os.Getenv("TWITCH_TOKEN")
)
