package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

/*
 * @module presence
 * @module_desc Updates the presence status of the bot to display the next stream
 */

const (
	presenceTimeDay = 24 * time.Hour
)

func init() {
	RegisterModule("presence", func() module { return &modPresence{} })
}

type modPresence struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string
}

func (m modPresence) ID() string { return m.id }

func (m *modPresence) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

	if err := attrs.Expect(
		"fallback_text",
		"twitch_channel_id",
		"twitch_client_id",
		"twitch_client_secret",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	// @attr cron optional string "* * * * *" When to execute the module
	if _, err := crontab.AddFunc(attrs.MustString("cron", ptrString("* * * * *")), m.cronUpdatePresence); err != nil {
		return errors.Wrap(err, "adding cron function")
	}

	return nil
}

func (m modPresence) Setup() error { return nil }

func (m modPresence) cronUpdatePresence() {
	var nextStream *time.Time

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

	for _, seg := range data.Data.Segments {
		if seg.StartTime == nil || seg.CanceledUntil != nil {
			continue
		}

		if seg.StartTime.Before(time.Now()) {
			continue
		}

		nextStream = seg.StartTime
		break
	}

	// @attr fallback_text required string "" What to set the text to when no stream is found (`playing <text>`)
	status := m.attrs.MustString("fallback_text", nil)
	if nextStream != nil {
		status = m.durationToHumanReadable(time.Since(*nextStream))
	}

	if err := m.discord.UpdateGameStatus(0, status); err != nil {
		log.WithError(err).Error("Unable to update status")
	}

	log.Debug("Updated presence")
}

func (m modPresence) durationToHumanReadable(d time.Duration) string {
	d = time.Duration(math.Abs(float64(d)))

	if d > presenceTimeDay {
		return fmt.Sprintf("in %.0f Tagen", math.Round(float64(d)/float64(presenceTimeDay)))
	}

	if d > time.Hour {
		return fmt.Sprintf("in %.0f Stunden", math.Round(float64(d)/float64(time.Hour)))
	}

	if d > time.Minute {
		return fmt.Sprintf("in %.0f Minuten", math.Round(float64(d)/float64(time.Minute)))
	}

	return "gleich"
}
