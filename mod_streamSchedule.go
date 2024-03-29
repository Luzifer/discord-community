package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
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
		"twitch_channel_id",
		"twitch_client_id",
		"twitch_client_secret",
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

//nolint:funlen,gocyclo // Seeing no sense to split for 5 lines
func (m modStreamSchedule) cronUpdateSchedule() {
	twitch := newTwitchAdapter(
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		m.attrs.MustString("twitch_client_id", nil),
		// @attr twitch_client_secret required string "" Secret for the Twitch app identified with twitch_client_id
		m.attrs.MustString("twitch_client_secret", nil),
		"", // No User-Token used
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

	var msgEmbed *discordgo.MessageEmbed
	// @attr embed_title optional string "" Title of the embed (embed will not be added when title is missing)
	if m.attrs.MustString("embed_title", ptrStringEmpty) != "" {
		msgEmbed = m.assembleEmbed(data)
	}

	var contentString string
	// @attr content optional string "" Message content to post above the embed - Allows Go templating, make sure to proper escape the template strings. See [here](https://github.com/Luzifer/discord-community/blob/5f004fdab066f16580f41076a4e6d8668fe743c9/twitch.go#L53-L71) for available data object.
	if m.attrs.MustString("content", ptrStringEmpty) != "" {
		if contentString, err = m.executeContentTemplate(data); err != nil {
			log.WithError(err).Error("executing stream schedule template")
			return
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
		var oldEmbed *discordgo.MessageEmbed
		if len(managedMsg.Embeds) > 0 {
			oldEmbed = managedMsg.Embeds[0]
		}

		if isDiscordMessageEmbedEqual(oldEmbed, msgEmbed) && strings.TrimSpace(managedMsg.Content) == strings.TrimSpace(contentString) {
			log.Debug("Stream Schedule is up-to-date")
			return
		}

		_, err = m.discord.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content: &contentString,
			Embed:   msgEmbed,

			ID:      managedMsg.ID,
			Channel: channelID,
		})
	} else {
		managedMsg, err = m.discord.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: contentString,
			Embed:   msgEmbed,
		})
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

func (m modStreamSchedule) assembleEmbed(data *twitchStreamSchedule) *discordgo.MessageEmbed {
	msgEmbed := &discordgo.MessageEmbed{
		// @attr embed_color optional int64 "0x2ECC71" Integer / HEX representation of the color for the embed
		Color: int(m.attrs.MustInt64("embed_color", ptrInt64(streamScheduleDefaultColor))),
		// @attr embed_description optional string "" Description for the embed block
		Description: strings.TrimSpace(m.attrs.MustString("embed_description", ptrStringEmpty)),
		Fields:      []*discordgo.MessageEmbedField{},
		Timestamp:   time.Now().Format(time.RFC3339),
		Title:       m.attrs.MustString("embed_title", nil),
		Type:        discordgo.EmbedTypeRich,
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
		switch {
		case seg.StartTime == nil || seg.CanceledUntil != nil:
			// No start-time: We skip this entry
			continue

		case seg.Category != nil && seg.Title == "":
			// No title but category set: use category as title
			title = seg.Category.Name

		case seg.Category != nil && !strings.Contains(seg.Title, seg.Category.Name):
			// Title and category set but category not part of title: Add it in braces
			title = fmt.Sprintf("%s (%s)", seg.Title, seg.Category.Name)

		case seg.Category == nil && seg.Title == "":
			// Unnamed stream without category: don't display empty field
			continue
		}

		msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   m.formatTime(*seg.StartTime),
			Value:  strings.TrimSpace(title),
			Inline: false,
		})

		// @attr schedule_entries optional int64 "5" How many schedule entries to add to the embed as fields
		if len(msgEmbed.Fields) == int(m.attrs.MustInt64("schedule_entries", defaultStreamScheduleEntries)) {
			break
		}
	}

	return msgEmbed
}

func (m modStreamSchedule) executeContentTemplate(data *twitchStreamSchedule) (string, error) {
	fns := sprig.FuncMap()
	fns["formatTime"] = m.formatTime

	tpl, err := template.New("streamschedule").
		Funcs(fns).
		Parse(m.attrs.MustString("content", ptrStringEmpty))
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, data.Data)
	return buf.String(), errors.Wrap(err, "executing template")
}

func (m modStreamSchedule) formatTime(t time.Time) string {
	// @attr timezone optional string "UTC" Timezone to display the times in (e.g. `Europe/Berlin`)
	tz, err := time.LoadLocation(m.attrs.MustString("timezone", ptrString("UTC")))
	if err != nil {
		log.WithError(err).Fatal("Unable to load timezone")
	}

	return localeStrftime(
		t.In(tz),
		// @attr time_format optional string "%b %d, %Y %I:%M %p" Time format in [limited strftime format](https://github.com/Luzifer/discord-community/blob/master/strftime.go) to use (e.g. `%a. %d.%m. %H:%M Uhr`)
		m.attrs.MustString("time_format", ptrString("%b %d, %Y %I:%M %p")),
		// @attr locale optional string "en_US" Locale to translate the date to ([supported locales](https://github.com/goodsign/monday/blob/24c0b92f25dca51152defe82cefc7f7fc1c92009/locale.go#L9-L49))
		m.attrs.MustString("locale", ptrString("en_US")),
	)
}
