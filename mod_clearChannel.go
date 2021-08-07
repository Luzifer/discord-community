package main

import (
	"sort"
	"strconv"
	"time"

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

/*
 * @module clearchannel
 * @module_desc Cleans up old messages from a channel (for example announcement channel) which are older than the retention time
 */

const (
	clearChannelNumberOfMessagesToLoad = 100
)

func init() {
	RegisterModule("clearchannel", func() module { return &modClearChannel{} })
}

type modClearChannel struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string
}

func (m modClearChannel) ID() string { return m.id }

func (m *modClearChannel) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

	if err := attrs.Expect(
		"discord_channel_id",
		"retention",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	// @attr cron optional string "0 * * * *" When to execute the cleaner
	if _, err := crontab.AddFunc(attrs.MustString("cron", ptrString("0 * * * *")), m.cronClearChannel); err != nil {
		return errors.Wrap(err, "adding cron function")
	}

	return nil
}

func (m modClearChannel) Setup() error { return nil }

func (m modClearChannel) cronClearChannel() {
	var (
		after        = "0"
		err          error
		onlyUsers    []string
		protectUsers []string

		// @attr discord_channel_id required string "" ID of the Discord channel to clean up
		channelID = m.attrs.MustString("discord_channel_id", nil)
		// @attr retention required duration "" How long to keep messages in this channel
		retention = m.attrs.MustDuration("retention", nil)
	)

	// @attr only_users optional []string "[]" When this list contains user IDs, only posts authored by those IDs will be deleted
	onlyUsers, err = m.attrs.StringSlice("only_users")
	switch err {
	case nil, errValueNotSet:
		// This is fine
	default:
		log.WithError(err).Error("Unable to load value for only_users")
		return
	}

	// @attr protect_users optional []string "[]" When this list contains user IDs, posts authored by those IDs will not be deleted
	protectUsers, err = m.attrs.StringSlice("protect_users")
	switch err {
	case nil, errValueNotSet:
		// This is fine
	default:
		log.WithError(err).Error("Unable to load value for protect_users")
		return
	}

	for {
		msgs, err := m.discord.ChannelMessages(channelID, clearChannelNumberOfMessagesToLoad, "", after, "")
		if err != nil {
			log.WithError(err).Error("Unable to fetch announcement channel messages")
			return
		}

		sort.Slice(msgs, func(i, j int) bool {
			iu, _ := strconv.ParseUint(msgs[i].ID, 10, 64) //nolint: gomnd // These make no sense to define as constants
			ju, _ := strconv.ParseUint(msgs[j].ID, 10, 64) //nolint: gomnd // These make no sense to define as constants
			return iu < ju
		})

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			mt, err := msg.Timestamp.Parse()
			if err != nil {
				log.WithField("msg_id", msg.ID).WithError(err).Error("Unable to parse message timestamp")
				break
			}

			if time.Since(mt) < retention {
				// We got to the first message within the retention time, we can end now
				break
			}

			if len(onlyUsers) > 0 && !str.StringInSlice(msg.Author.ID, onlyUsers) {
				// Is not written by one of the users we may purge
				continue
			}

			if len(protectUsers) > 0 && str.StringInSlice(msg.Author.ID, protectUsers) {
				// Is written by protected user, we may not purge
				continue
			}

			if err = m.discord.ChannelMessageDelete(channelID, msg.ID); err != nil {
				log.WithError(err).Error("Unable to delete messages")
				return
			}

			after = msg.ID
		}
	}
}
