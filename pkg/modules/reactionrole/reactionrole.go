// Package reactionrole implements a module for Discord reaction-based role assignment.
package reactionrole

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/env"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	"github.com/Luzifer/discord-community/pkg/config"
	"github.com/Luzifer/discord-community/pkg/helpers"
	"github.com/Luzifer/discord-community/pkg/modules"
)

/*
 * @module reactionrole
 * @module_desc Creates a post with pre-set reactions and assigns roles on reaction
 */

type modReactionRole struct {
	attrs   attributestore.ModuleAttributeStore
	discord *discordgo.Session
	id      string
	config  *config.File
	store   *modules.MetaStore
}

func init() {
	modules.RegisterModule("reactionrole", func() modules.Module { return &modReactionRole{} })
}

func (m modReactionRole) ID() string { return m.id }

func (m *modReactionRole) Initialize(args modules.ModuleInitArgs) error {
	m.attrs = args.Attrs
	m.discord = args.Discord
	m.id = args.ID
	m.store = args.Store
	m.config = args.Config

	if err := m.attrs.Expect(
		"discord_channel_id",
		"reaction_roles",
	); err != nil {
		return fmt.Errorf("validating attributes: %w", err)
	}

	m.discord.AddHandler(m.handleMessageReactionAdd)
	m.discord.AddHandler(m.handleMessageReactionRemove)

	return nil
}

//nolint:funlen,gocognit,gocyclo // Single task, seeing no sense in splitting
func (m modReactionRole) Setup() error {
	var err error

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	channelID := m.attrs.MustString("discord_channel_id", nil)

	// @attr content optional string "" Message content to post above the embed
	contentString := m.attrs.MustString("content", new(""))

	var msgEmbed *discordgo.MessageEmbed
	// @attr embed_title optional string "" Title of the embed (embed will not be added when title is missing)
	if title := m.attrs.MustString("embed_title", new("")); title != "" {
		msgEmbed = &discordgo.MessageEmbed{
			// @attr embed_color optional int64 "0x2ECC71" Integer / HEX representation of the color for the embed
			Color: int(m.attrs.MustInt64("embed_color", helpers.StreamScheduleDefaultColor)),
			// @attr embed_description optional string "" Description for the embed block
			Description: strings.TrimSpace(m.attrs.MustString("embed_description", new(""))),
			Timestamp:   time.Now().Format(time.RFC3339),
			Title:       title,
			Type:        discordgo.EmbedTypeRich,
		}

		if m.attrs.MustString("embed_thumbnail_url", new("")) != "" {
			msgEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				// @attr embed_thumbnail_url optional string "" Publically hosted image URL to use as thumbnail
				URL: m.attrs.MustString("embed_thumbnail_url", new("")),
				// @attr embed_thumbnail_width optional int64 "" Width of the thumbnail
				Width: int(m.attrs.MustInt64("embed_thumbnail_width", new(int64(0)))),
				// @attr embed_thumbnail_height optional int64 "" Height of the thumbnail
				Height: int(m.attrs.MustInt64("embed_thumbnail_height", new(int64(0)))),
			}
		}
	}

	reactionListRaw, err := m.attrs.StringSlice("reaction_roles")
	if err != nil {
		return fmt.Errorf("getting role list: %w", err)
	}
	var reactionList []string
	for _, r := range reactionListRaw {
		reactionList = append(reactionList, strings.Split(r, "=")[0])
	}

	var managedMsg *discordgo.Message
	if err = m.store.ReadWithLock(m.id, func(a attributestore.ModuleAttributeStore) error {
		mid, err := a.String("message_id")
		if err == attributestore.ErrValueNotSet {
			return nil
		}

		if managedMsg, err = m.discord.ChannelMessage(channelID, mid); err != nil {
			return fmt.Errorf("fetching managed message: %w", err)
		}

		return nil
	}); err != nil && !strings.Contains(err.Error(), "404") {
		return fmt.Errorf("getting managed message: %w", err)
	}

	if managedMsg == nil {
		managedMsg, err = m.discord.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: contentString,
			Embed:   msgEmbed,
		})
	} else if (len(managedMsg.Embeds) > 0 && !helpers.IsDiscordMessageEmbedEqual(managedMsg.Embeds[0], msgEmbed)) || managedMsg.Content != contentString {
		_, err = m.discord.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content: &contentString,
			Embed:   msgEmbed,

			ID:      managedMsg.ID,
			Channel: channelID,
		})
	}
	if err != nil {
		return fmt.Errorf("updating / creating message: %w", err)
	}

	if err = m.store.Set(m.id, "message_id", managedMsg.ID); err != nil {
		return fmt.Errorf("storing managed message id: %w", err)
	}

	var addedReactions []string

	for _, r := range managedMsg.Reactions {
		okName := slices.Contains(reactionList, r.Emoji.Name)

		compiledName := fmt.Sprintf(":%s:%s", r.Emoji.Name, r.Emoji.ID)
		okCode := slices.Contains(reactionList, compiledName)

		if !okCode && !okName {
			id := r.Emoji.ID
			if id == "" {
				id = r.Emoji.Name
			}

			if err = m.discord.MessageReactionsRemoveEmoji(channelID, managedMsg.ID, id); err != nil {
				return fmt.Errorf("removing reaction emoji: %w", err)
			}
			continue
		}

		addedReactions = append(addedReactions, compiledName, r.Emoji.Name)
	}

	for _, emoji := range reactionList {
		if !slices.Contains(addedReactions, emoji) {
			logrus.WithFields(logrus.Fields{
				"emote":   emoji,
				"message": managedMsg.ID,
				"module":  m.id,
			}).Trace("Adding emoji reaction")
			if err = m.discord.MessageReactionAdd(channelID, managedMsg.ID, emoji); err != nil {
				return fmt.Errorf("adding reaction emoji: %w", err)
			}
		}
	}

	return nil
}

func (m modReactionRole) extractRoles() (map[string]string, error) {
	// @attr reaction_roles required []string "" List of strings in format `emote=role-id[:set]`. `emote` equals an unicode emote (✅) or a custom emote in form `:<emote-name>:<emote-id>`. `role-id` is the integer ID of the guilds role to add with this emote. If `:set` is added at the end, the role will only be added but not removed when the reaction is removed.
	list, err := m.attrs.StringSlice("reaction_roles")
	if err != nil {
		return nil, fmt.Errorf("getting role list: %w", err)
	}

	return env.ListToMap(list), nil
}

//revive:disable-next-line:flag-parameter // not a flag, just telling whether a reaction was added or removed
func (m modReactionRole) handleMessageReaction(_ *discordgo.Session, e *discordgo.MessageReaction, add bool) {
	if e.UserID == m.discord.State.User.ID {
		// Reaction was manipulated by the bot, ignore
		return
	}

	var (
		err       error
		messageID string
	)

	if err = m.store.ReadWithLock(m.id, func(a attributestore.ModuleAttributeStore) error {
		if messageID, err = a.String("message_id"); err != nil {
			return fmt.Errorf("reading message ID: %w", err)
		}

		return nil
	}); err != nil {
		logrus.WithError(err).Error("Unable to get managed message ID")
		return
	}

	if messageID == "" || e.MessageID != messageID {
		// This is not our managed message (or we don't have one), we don't care
		return
	}

	roles, err := m.extractRoles()
	if err != nil {
		logrus.WithError(err).Error("Unable to extract role mapping")
		return
	}

	for _, check := range []string{e.Emoji.Name, fmt.Sprintf(":%s:%s", e.Emoji.Name, e.Emoji.ID)} {
		role, ok := roles[check]
		if !ok {
			continue
		}

		if add {
			if err = m.discord.GuildMemberRoleAdd(m.config.GuildID, e.UserID, strings.Split(role, ":")[0]); err != nil {
				logrus.WithError(err).Error("Unable to add role to user")
			}
			return
		}

		if !strings.HasSuffix(role, ":set") {
			if err = m.discord.GuildMemberRoleRemove(m.config.GuildID, e.UserID, strings.Split(role, ":")[0]); err != nil {
				logrus.WithError(err).Error("Unable to remove role to user")
			}
			return
		}
	}
}

func (m modReactionRole) handleMessageReactionAdd(d *discordgo.Session, e *discordgo.MessageReactionAdd) {
	m.handleMessageReaction(d, e.MessageReaction, true)
}

func (m modReactionRole) handleMessageReactionRemove(d *discordgo.Session, e *discordgo.MessageReactionRemove) {
	m.handleMessageReaction(d, e.MessageReaction, false)
}
