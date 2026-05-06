// Package clearchannel implements a module for deleting old Discord messages.
package clearchannel

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	"github.com/Luzifer/discord-community/pkg/modules"
)

/*
 * @module clearchannel
 * @module_desc Cleans up old messages from a channel (for example announcement channel) which are older than the retention time
 */

const (
	clearChannelNumberOfMessagesToLoad = 100
)

type modClearChannel struct {
	attrs   attributestore.ModuleAttributeStore
	discord *discordgo.Session
	id      string
}

func init() {
	modules.RegisterModule("clearchannel", func() modules.Module { return &modClearChannel{} })
}

func (m modClearChannel) ID() string { return m.id }

func (m *modClearChannel) Initialize(args modules.ModuleInitArgs) error {
	m.attrs = args.Attrs
	m.discord = args.Discord
	m.id = args.ID

	if err := args.Attrs.Expect(
		"discord_channel_id",
		"retention",
	); err != nil {
		return fmt.Errorf("validating attributes: %w", err)
	}

	// @attr cron optional string "0 * * * *" When to execute the cleaner
	if _, err := args.Crontab.AddFunc(args.Attrs.MustString("cron", new("0 * * * *")), m.cronClearChannel); err != nil {
		return fmt.Errorf("adding cron function: %w", err)
	}

	return nil
}

func (modClearChannel) Setup() error { return nil }

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
	case nil, attributestore.ErrValueNotSet:
		// This is fine
	default:
		logrus.WithError(err).Error("Unable to load value for only_users")
		return
	}

	// @attr protect_users optional []string "[]" When this list contains user IDs, posts authored by those IDs will not be deleted
	protectUsers, err = m.attrs.StringSlice("protect_users")
	switch err {
	case nil, attributestore.ErrValueNotSet:
		// This is fine
	default:
		logrus.WithError(err).Error("Unable to load value for protect_users")
		return
	}

	for {
		msgs, err := m.discord.ChannelMessages(channelID, clearChannelNumberOfMessagesToLoad, "", after, "")
		if err != nil {
			logrus.WithError(err).Error("Unable to fetch announcement channel messages")
			return
		}

		sort.Slice(msgs, func(i, j int) bool {
			iu, _ := strconv.ParseUint(msgs[i].ID, 10, 64)
			ju, _ := strconv.ParseUint(msgs[j].ID, 10, 64)
			return iu < ju
		})

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			if time.Since(msg.Timestamp) < retention {
				// We got to the first message within the retention time, we can end now
				break
			}

			if len(onlyUsers) > 0 && !slices.Contains(onlyUsers, msg.Author.ID) {
				// Is not written by one of the users we may purge
				continue
			}

			if len(protectUsers) > 0 && slices.Contains(protectUsers, msg.Author.ID) {
				// Is written by protected user, we may not purge
				continue
			}

			if err = m.discord.ChannelMessageDelete(channelID, msg.ID); err != nil {
				logrus.WithError(err).Error("Unable to delete messages")
				return
			}

			after = msg.ID
		}
	}
}
