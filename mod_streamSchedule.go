package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/v2/backoff"
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
	twitchAPIRequestLimit   = 5
	twitchAPIRequestTimeout = 2 * time.Second
)

var (
	defaultStreamScheduleEntries  = ptrInt64(5)
	defaultStreamSchedulePastTime = ptrDuration(15 * time.Minute)
)

func init() {
	RegisterModule("schedule", func() module { return &modStreamSchedule{} })
}

type (
	modStreamSchedule struct {
		attrs   moduleAttributeStore
		discord *discordgo.Session
	}

	twitchStreamScheduleResponse struct {
		Data struct {
			Segments []struct {
				ID            string     `json:"id"`
				StartTime     *time.Time `json:"start_time"`
				EndTime       *time.Time `json:"end_time"`
				Title         string     `json:"title"`
				CanceledUntil *time.Time `json:"canceled_until"`
				Category      *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				IsRecurring bool `json:"is_recurring"`
			} `json:"segments"`
			BroadcasterID    string `json:"broadcaster_id"`
			BroadcasterName  string `json:"broadcaster_name"`
			BroadcasterLogin string `json:"broadcaster_login"`
			Vacation         *struct {
				StartTime *time.Time `json:"start_time"`
				EndTime   *time.Time `json:"end_time"`
			} `json:"vacation"`
		} `json:"data"`
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}
)

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

func (m modStreamSchedule) cronUpdateSchedule() {
	var data twitchStreamScheduleResponse
	if err := backoff.NewBackoff().WithMaxIterations(twitchAPIRequestLimit).Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), twitchAPIRequestTimeout)
		defer cancel()

		u, _ := url.Parse("https://api.twitch.tv/helix/schedule")
		params := make(url.Values)
		// @attr twitch_channel_id required string "" ID (not name) of the channel to fetch the schedule from
		params.Set("broadcaster_id", m.attrs.MustString("twitch_channel_id", nil))
		// @attr schedule_past_time optional duration "15m" How long in the past should the schedule contain an entry
		params.Set("start_time", time.Now().Add(-m.attrs.MustDuration("schedule_past_time", defaultStreamSchedulePastTime)).Format(time.RFC3339))
		u.RawQuery = params.Encode()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		// @attr twitch_token required string "" Token for the user the `twitch_channel_id` belongs to
		req.Header.Set("Authorization", "Bearer "+m.attrs.MustString("twitch_token", nil))
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		req.Header.Set("Client-Id", m.attrs.MustString("twitch_client_id", nil))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "fetching schedule")
		}
		defer resp.Body.Close()

		return errors.Wrap(
			json.NewDecoder(resp.Body).Decode(&data),
			"decoding schedule response",
		)
	}); err != nil {
		log.WithError(err).Error("Unable to fetch stream schedule")
		return
	}

	msgEmbed := &discordgo.MessageEmbed{
		// @attr embed_color optional int64 "3066993" Integer representation of the hex color for the embed (default is #2ECC71)
		Color: int(m.attrs.MustInt64("embed_color", ptrInt64(3066993))),
		// @attr embed_description optional string "" Description for the embed block
		Description: m.attrs.MustString("embed_description", ptrStringEmpty),
		Fields:      []*discordgo.MessageEmbedField{},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			// @attr embed_thumbnail_url optional string "" Publically hosted image URL to use as thumbnail
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
	msgs, err := m.discord.ChannelMessages(m.attrs.MustString("discord_channel_id", nil), 100, "", "", "")
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
