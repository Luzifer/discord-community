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
	streamScheduleDefaultColor           = 0x2ECC71
	streamScheduleNumberOfMessagesToLoad = 100
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
}

func (m *modStreamSchedule) Initialize(crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord

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

	msgEmbed := &discordgo.MessageEmbed{
		// @attr embed_color optional int64 "0x2ECC71" Integer representation of the hex color for the embed
		Color: int(m.attrs.MustInt64("embed_color", ptrInt64(streamScheduleDefaultColor))),
		// @attr embed_description optional string "" Description for the embed block
		Description: m.attrs.MustString("embed_description", ptrStringEmpty),
		Fields:      []*discordgo.MessageEmbedField{},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			// @attr embed_thumbnail_url optional string "" Publically hosted image URL to u100se as thumbnail
			URL: m.attrs.MustString("embed_thumbnail_url", ptrStringEmpty),
			// @attr embed_thumbnail_width optional int64 "" Width of the thumbnail
			Width: int(m.attrs.MustInt64("embed_thumbnail_width", ptrInt64Zero)),
			// @attr embed_thumbnail_height optional int64 "" Height of the thumbnail
			Height: int(m.attrs.MustInt64("embed_thumbnail_height", ptrInt64Zero)),
		},
		Timestamp: time.Now().Format(time.RFC3339),
		// @attr embed_title required string "" Title of the embed (used to find the managed post, must be unique for that channel)
		Title: m.attrs.MustString("embed_title", nil),
		Type:  discordgo.EmbedTypeRich,
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

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	msgs, err := m.discord.ChannelMessages(m.attrs.MustString("discord_channel_id", nil), streamScheduleNumberOfMessagesToLoad, "", "", "")
	if err != nil {
		log.WithError(err).Error("Unable to fetch announcement channel messages")
		return
	}

	var managedMsg *discordgo.Message
	for _, msg := range msgs {
		if len(msg.Embeds) == 0 || msg.Embeds[0].Title != msgEmbed.Title {
			continue
		}

		managedMsg = msg
	}

	if managedMsg != nil {
		oldEmbed := managedMsg.Embeds[0]

		if !m.embedNeedsUpdate(oldEmbed, msgEmbed) {
			log.Debug("Stream Schedule is up-to-date")
			return
		}

		_, err = m.discord.ChannelMessageEditEmbed(m.attrs.MustString("discord_channel_id", nil), managedMsg.ID, msgEmbed)
	} else {
		_, err = m.discord.ChannelMessageSendEmbed(m.attrs.MustString("discord_channel_id", nil), msgEmbed)
	}
	if err != nil {
		log.WithError(err).Error("Unable to announce streamplan")
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

func (m modStreamSchedule) embedNeedsUpdate(o, n *discordgo.MessageEmbed) bool {
	if o.Title != n.Title {
		return true
	}

	if o.Description != n.Description {
		return true
	}

	if o.Thumbnail != nil && n.Thumbnail == nil || o.Thumbnail == nil && n.Thumbnail != nil {
		return true
	}

	if o.Thumbnail != nil && o.Thumbnail.URL != n.Thumbnail.URL {
		return true
	}

	if len(o.Fields) != len(n.Fields) {
		return true
	}

	for i := range o.Fields {
		if o.Fields[i].Name != n.Fields[i].Name {
			return true
		}

		if o.Fields[i].Value != n.Fields[i].Value {
			return true
		}

		if o.Fields[i].Inline != n.Fields[i].Inline {
			return true
		}
	}

	return false
}
