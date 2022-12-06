package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/go_helpers/v2/env"
	"github.com/Luzifer/go_helpers/v2/str"
)

/*
 * @module reactionrole
 * @module_desc Creates a post with pre-set reactions and assigns roles on reaction
 */

func init() {
	RegisterModule("reactionrole", func() module { return &modReactionRole{} })
}

type modReactionRole struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string
}

func (m modReactionRole) ID() string { return m.id }

func (m *modReactionRole) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

	if err := attrs.Expect(
		"discord_channel_id",
		"reaction_roles",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	discord.AddHandler(m.handleMessageReactionAdd)
	discord.AddHandler(m.handleMessageReactionRemove)

	return nil
}

//nolint:funlen,gocyclo // Single task, seeing no sense in splitting
func (m modReactionRole) Setup() error {
	var err error

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	channelID := m.attrs.MustString("discord_channel_id", nil)

	// @attr content optional string "" Message content to post above the embed
	contentString := m.attrs.MustString("content", ptrStringEmpty)

	var msgEmbed *discordgo.MessageEmbed
	// @attr embed_title optional string "" Title of the embed (embed will not be added when title is missing)
	if title := m.attrs.MustString("embed_title", ptrStringEmpty); title != "" {
		msgEmbed = &discordgo.MessageEmbed{
			// @attr embed_color optional int64 "0x2ECC71" Integer / HEX representation of the color for the embed
			Color: int(m.attrs.MustInt64("embed_color", ptrInt64(streamScheduleDefaultColor))),
			// @attr embed_description optional string "" Description for the embed block
			Description: strings.TrimSpace(m.attrs.MustString("embed_description", ptrStringEmpty)),
			Timestamp:   time.Now().Format(time.RFC3339),
			Title:       title,
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
	}

	reactionListRaw, err := m.attrs.StringSlice("reaction_roles")
	if err != nil {
		return errors.Wrap(err, "getting role list")
	}
	var reactionList []string
	for _, r := range reactionListRaw {
		reactionList = append(reactionList, strings.Split(r, "=")[0])
	}

	var managedMsg *discordgo.Message
	if err = store.ReadWithLock(m.id, func(a moduleAttributeStore) error {
		mid, err := a.String("message_id")
		if err == errValueNotSet {
			return nil
		}

		managedMsg, err = m.discord.ChannelMessage(channelID, mid)
		return errors.Wrap(err, "fetching managed message")
	}); err != nil && !strings.Contains(err.Error(), "404") {
		return err
	}

	if managedMsg == nil {
		managedMsg, err = m.discord.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: contentString,
			Embed:   msgEmbed,
		})
	} else if !isDiscordMessageEmbedEqual(managedMsg.Embeds[0], msgEmbed) || managedMsg.Content != contentString {
		_, err = m.discord.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Content: &contentString,
			Embed:   msgEmbed,

			ID:      managedMsg.ID,
			Channel: channelID,
		})
	}
	if err != nil {
		return errors.Wrap(err, "updating / creating message")
	}

	if err = store.Set(m.id, "message_id", managedMsg.ID); err != nil {
		return errors.Wrap(err, "storing managed message id")
	}

	var addedReactions []string

	for _, r := range managedMsg.Reactions {
		okName := str.StringInSlice(r.Emoji.Name, reactionList)

		compiledName := fmt.Sprintf(":%s:%s", r.Emoji.Name, r.Emoji.ID)
		okCode := str.StringInSlice(compiledName, reactionList)

		if !okCode && !okName {
			id := r.Emoji.ID
			if id == "" {
				id = r.Emoji.Name
			}

			if err = m.discord.MessageReactionsRemoveEmoji(channelID, managedMsg.ID, id); err != nil {
				return errors.Wrap(err, "removing reaction emoji")
			}
			continue
		}

		addedReactions = append(addedReactions, compiledName, r.Emoji.Name)
	}

	for _, emoji := range reactionList {
		if !str.StringInSlice(emoji, addedReactions) {
			log.WithFields(log.Fields{
				"emote":   emoji,
				"message": managedMsg.ID,
				"module":  m.id,
			}).Trace("Adding emoji reaction")
			if err = m.discord.MessageReactionAdd(channelID, managedMsg.ID, emoji); err != nil {
				return errors.Wrap(err, "adding reaction emoji")
			}
		}
	}

	return nil
}

func (m modReactionRole) extractRoles() (map[string]string, error) {
	// @attr reaction_roles required []string "" List of strings in format `emote=role-id[:set]`. `emote` equals an unicode emote (✅) or a custom emote in form `:<emote-name>:<emote-id>`. `role-id` is the integer ID of the guilds role to add with this emote. If `:set` is added at the end, the role will only be added but not removed when the reaction is removed.
	list, err := m.attrs.StringSlice("reaction_roles")
	if err != nil {
		return nil, errors.Wrap(err, "getting role list")
	}

	return env.ListToMap(list), nil
}

func (m modReactionRole) handleMessageReaction(d *discordgo.Session, e *discordgo.MessageReaction, add bool) {
	if e.UserID == m.discord.State.User.ID {
		// Reaction was manipulated by the bot, ignore
		return
	}

	var (
		err       error
		messageID string
	)

	if err = store.ReadWithLock(m.id, func(a moduleAttributeStore) error {
		messageID, err = a.String("message_id")
		return errors.Wrap(err, "reading message ID")
	}); err != nil {
		log.WithError(err).Error("Unable to get managed message ID")
		return
	}

	if messageID == "" || e.MessageID != messageID {
		// This is not our managed message (or we don't have one), we don't care
		return
	}

	roles, err := m.extractRoles()
	if err != nil {
		log.WithError(err).Error("Unable to extract role mapping")
		return
	}

	for _, check := range []string{e.Emoji.Name, fmt.Sprintf(":%s:%s", e.Emoji.Name, e.Emoji.ID)} {
		role, ok := roles[check]
		if !ok {
			continue
		}

		if add {
			if err = m.discord.GuildMemberRoleAdd(config.GuildID, e.UserID, strings.Split(role, ":")[0]); err != nil {
				log.WithError(err).Error("Unable to add role to user")
			}
			return
		}

		if !strings.HasSuffix(role, ":set") {
			if err = m.discord.GuildMemberRoleRemove(config.GuildID, e.UserID, strings.Split(role, ":")[0]); err != nil {
				log.WithError(err).Error("Unable to remove role to user")
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
