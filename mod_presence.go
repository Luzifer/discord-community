package main

import (
	"fmt"
	"math"
	"strings"
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

func init() {
	RegisterModule("presence", func() module { return &modPresence{} })
}

type modPresence struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
}

func (m *modPresence) Initialize(crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord

	if err := attrs.Expect(
		"fallback_text",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	// @attr cron optional string "* * * * *" When to execute the module
	if _, err := crontab.AddFunc(attrs.MustString("cron", ptrString("* * * * *")), m.cronUpdatePresence); err != nil {
		return errors.Wrap(err, "adding cron function")
	}

	return nil
}

func (m modPresence) cronUpdatePresence() {
	var nextStream *time.Time = nil

	// FIXME: Get next stream status

	// @attr fallback_text required string "" What to set the text to when no stream is found (`playing <text>`)
	status := m.attrs.MustString("fallback_text", nil)
	if nextStream != nil {
		status = fmt.Sprintf("in: %s", m.durationToHumanReadable(time.Since(*nextStream)))
	}

	if err := m.discord.UpdateGameStatus(0, status); err != nil {
		log.WithError(err).Error("Unable to update status")
	}

	log.Debug("Updated presence")
}

func (m modPresence) durationToHumanReadable(d time.Duration) string {
	var elements []string

	d = time.Duration(math.Abs(float64(d)))
	for div, req := range map[time.Duration]bool{
		time.Hour * 24: false,
		time.Hour:      true,
		time.Minute:    true,
	} {
		if d < div && !req {
			continue
		}

		pt := d / div
		d -= pt * div
		elements = append(elements, fmt.Sprintf("%.2d", pt))
	}

	return strings.Join(elements, ":")
}
