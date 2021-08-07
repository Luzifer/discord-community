package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

/*
 * @module schedule
 * @module_desc Posts stream schedule derived from Twitch schedule as embed in Discord channel
 */

const (
	streamScheduleDefaultColor = 0x2ECC71
)

var (
	defaultStreamScheduleEntries  = ptrInt64(5)                   //nolint: gomnd // This is already the "constant"
	defaultStreamSchedulePastTime = ptrDuration(15 * time.Minute) //nolint: gomnd // This is already the "constant"
)

func init() {
	RegisterModule("schedule", func() module { return &modStreamSchedule{} })
}

type modStreamSchedule struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string
}

func (m modStreamSchedule) ID() string { return m.id }

func (m *modStreamSchedule) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

	if err := attrs.Expect(
		"discord_channel_id",
		"embed_title",
		"twitch_channel_id",
		"twitch_client_id",
		"twitch_token",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	// @attr cron optional string "*/10 * * * *" When to execute the schedule transfer
	if _, err := crontab.AddFunc(attrs.MustString("cron", ptrString("*/10 * * * *")), m.cronUpdateSchedule); err != nil {
		return errors.Wrap(err, "adding cron function")
	}

	return nil
}

func (m modStreamSchedule) Setup() error { return nil }

func (m modStreamSchedule) cronUpdateSchedule() {
	twitch := newTwitchAdapter(
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		m.attrs.MustString("twitch_client_id", nil),
		"", // No Client Secret used
		// @attr twitch_token required string "" Token for the user the `twitch_channel_id` belongs to
		m.attrs.MustString("twitch_token", nil),
	)

	data, err := twitch.GetChannelStreamSchedule(
		context.Background(),
		// @attr twitch_channel_id required string "" ID (not name) of the channel to fetch the schedule from
		m.attrs.MustString("twitch_channel_id", nil),
		// @attr schedule_past_time optional duration "15m" How long in the past should the schedule contain an entry
		ptrTime(time.Now().Add(-m.attrs.MustDuration("schedule_past_time", defaultStreamSchedulePastTime))),
	)
	if err != nil {
		log.WithError(err).Error("Unable to fetch stream schedule")
		return
	}

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	channelID := m.attrs.MustString("discord_channel_id", nil)

	msgEmbed := &discordgo.MessageEmbed{
		// @attr embed_color optional int64 "0x2ECC71" Integer representation of the hex color for the embed
		Color: int(m.attrs.MustInt64("embed_color", ptrInt64(streamScheduleDefaultColor))),
		// @attr embed_description optional string "" Description for the embed block
		Description: strings.TrimSpace(m.attrs.MustString("embed_description", ptrStringEmpty)),
		Fields:      []*discordgo.MessageEmbedField{},
		Timestamp:   time.Now().Format(time.RFC3339),
		// @attr embed_title required string "" Title of the embed
		Title: m.attrs.MustString("embed_title", nil),
		Type:  discordgo.EmbedTypeRich,
	}

	if m.attrs.MustString("embed_thumbnail_url", ptrStringEmpty) != "" {
		msgEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			// @attr embed_thumbnail_url optional string "" Publically hosted image URL to use as thumbnail
			URL: m.attrs.MustString("embed_thumbnail_url", ptrStringEmpty),
			// @attr embed_thumbnail_width optional int64 "" Width of the thumbnail
			Width: int(m.attrs.MustInt64("embed_thumbnail_width", ptrInt64Zero)),
			// @attr embed_thumbnail_height optional int64 "" Height of the thumbnail
			Height: int(m.attrs.MustInt64("embed_thumbnail_height", ptrInt64Zero)),
		}
	}

	for _, seg := range data.Data.Segments {
		title := seg.Title
		if seg.Category != nil && seg.Category.Name != seg.Title {
			title = fmt.Sprintf("%s (%s)", seg.Title, seg.Category.Name)
		}

		if seg.StartTime == nil || seg.CanceledUntil != nil {
			continue
		}

		msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   m.formatGermanShort(*seg.StartTime),
			Value:  title,
			Inline: false,
		})

		// @attr schedule_entries optional int64 "5" How many schedule entries to add to the embed as fields
		if len(msgEmbed.Fields) == int(m.attrs.MustInt64("schedule_entries", defaultStreamScheduleEntries)) {
			break
		}
	}

	var managedMsg *discordgo.Message
	if err = store.ReadWithLock(m.id, func(a moduleAttributeStore) error {
		mid, err := a.String("message_id")
		if err == errValueNotSet {
			return nil
		}

		managedMsg, err = m.discord.ChannelMessage(channelID, mid)
		return errors.Wrap(err, "fetching managed message")
	}); err != nil {
		log.WithError(err).Error("Unable to fetch managed message for stream schedule")
		return
	}

	if managedMsg != nil {
		oldEmbed := managedMsg.Embeds[0]

		if isDiscordMessageEmbedEqual(oldEmbed, msgEmbed) {
			log.Debug("Stream Schedule is up-to-date")
			return
		}

		_, err = m.discord.ChannelMessageEditEmbed(channelID, managedMsg.ID, msgEmbed)
	} else {
		managedMsg, err = m.discord.ChannelMessageSendEmbed(channelID, msgEmbed)
	}
	if err != nil {
		log.WithError(err).Error("Unable to announce streamplan")
		return
	}

	if err = store.Set(m.id, "message_id", managedMsg.ID); err != nil {
		log.WithError(err).Error("Unable to store managed message id")
		return
	}

	log.Info("Updated Stream Schedule")
}

func (m modStreamSchedule) formatGermanShort(t time.Time) string {
	wd := map[time.Weekday]string{
		time.Monday:    "Mo.",
		time.Tuesday:   "Di.",
		time.Wednesday: "Mi.",
		time.Thursday:  "Do.",
		time.Friday:    "Fr.",
		time.Saturday:  "Sa.",
		time.Sunday:    "So.",
	}[t.Weekday()]

	tz, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.WithError(err).Fatal("Unable to load timezone Europe/Berlin")
	}

	return strings.Join([]string{wd, t.In(tz).Format("02.01. 15:04"), "Uhr"}, " ")
}
