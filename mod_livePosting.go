package main

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

/*
 * @module liveposting
 * @module_desc Announces stream live status based on Discord streaming status
 */

func init() {
	RegisterModule("liveposting", func() module { return &modLivePosting{} })
}

type modLivePosting struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
}

func (m *modLivePosting) Initialize(crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord

	if err := attrs.Expect(
		"discord_channel_id",
		"post_text",
		"whitelisted_role",
		"twitch_client_id",
		"twitch_client_secret",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	discord.AddHandler(m.handlePresenceUpdate)

	return nil
}

func (m modLivePosting) handlePresenceUpdate(d *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p.User == nil {
		// The frick? Non-user presence?
		return
	}

	logger := log.WithFields(log.Fields{
		"user": p.User.ID,
	})

	member, err := d.GuildMember(p.GuildID, p.User.ID)
	if err != nil {
		logger.WithError(err).Error("Unable to fetch member status for user")
		return
	}

	// @attr whitelisted_role optional string "" Only post for members of this role
	whitelistedRole := m.attrs.MustString("whitelisted_role", ptrStringEmpty)
	if whitelistedRole != "" && !str.StringInSlice(whitelistedRole, member.Roles) {
		// User is not allowed for this config
		return
	}

	var activity *discordgo.Activity

	for _, a := range p.Activities {
		if a.Type == discordgo.ActivityTypeStreaming {
			activity = a
			break
		}
	}

	if activity == nil {
		// No streaming activity: Do nothing
		return
	}

	u, err := url.Parse(activity.URL)
	if err != nil {
		logger.WithError(err).WithField("url", activity.URL).Warning("Unable to parse activity URL")
		return
	}

	if u.Host != "www.twitch.tv" {
		logger.WithError(err).WithField("url", activity.URL).Debug("Activity is not on Twitch")
		return
	}

	twitchUsername := strings.TrimLeft(u.Path, "/")

	twitch := newTwitchAdapter(
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		m.attrs.MustString("twitch_client_id", nil),
		// @attr twitch_client_secret required string "" Secret for the Twitch app identified with twitch_client_id
		m.attrs.MustString("twitch_client_secret", nil),
		"", // No User-Token used
	)

	users, err := twitch.GetUserByUsername(context.Background(), twitchUsername)
	if err != nil {
		logger.WithError(err).WithField("user", twitchUsername).Warning("Unable to fetch details for user")
		return
	}

	if l := len(users.Data); l != 1 {
		logger.WithError(err).WithField("url", activity.URL).Warning("Unable to fetch user for login")
		return
	}

	streams, err := twitch.GetStreamsForUser(context.Background(), twitchUsername)
	if err != nil {
		logger.WithError(err).WithField("user", twitchUsername).Debug("Unable to fetch streams for user")
		return
	}

	if l := len(streams.Data); l != 1 {
		logger.WithError(err).WithField("url", activity.URL).Debug("Unable to fetch streams for login")
		return
	}

	// @attr stream_freshness optional duration "5m" How long after stream start to post shoutout
	ignoreTime := m.attrs.MustDuration("stream_freshness", ptrDuration(5*time.Minute))
	if streams.Data[0].StartedAt.Add(ignoreTime).Before(time.Now()) {
		// Stream is too old, don't annoounce
		return
	}

	if err = m.sendLivePost(
		users.Data[0].Login,
		users.Data[0].DisplayName,
		streams.Data[0].Title,
		streams.Data[0].GameName,
		streams.Data[0].ThumbnailURL,
		users.Data[0].ProfileImageURL,
	); err != nil {
		logger.WithError(err).WithField("url", activity.URL).Error("Unable to send post")
		return
	}
}

func (m modLivePosting) sendLivePost(username, displayName, title, game, previewImage, profileImage string) error {
	postText := strings.NewReplacer(
		"${displayname}", displayName,
		"${username}", username,
	).Replace(
		// @attr post_text required string "" Message to post to channel use `${displayname}` and `${username}` as placeholders
		m.attrs.MustString("post_text", nil),
	)

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	msgs, err := m.discord.ChannelMessages(m.attrs.MustString("discord_channel_id", nil), 100, "", "", "")
	if err != nil {
		return errors.Wrap(err, "fetching previous messages")
	}

	ignoreTime := m.attrs.MustDuration("stream_freshness", ptrDuration(5*time.Minute))
	for _, msg := range msgs {
		mt, err := msg.Timestamp.Parse()
		if err != nil {
			return errors.Wrap(err, "parsing message timestamp")
		}
		if msg.Content == postText && time.Since(mt) < ignoreTime {
			return nil
		}
	}

	msgEmbed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    displayName,
			IconURL: profileImage,
		},
		Color: 0x6441a5,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Game", Value: game},
		},
		Image: &discordgo.MessageEmbedImage{
			URL:    strings.NewReplacer("{width}", "320", "{height}", "180").Replace(previewImage),
			Width:  320,
			Height: 180,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    profileImage,
			Width:  300,
			Height: 300,
		},
		Title: title,
		Type:  discordgo.EmbedTypeRich,
		URL:   strings.Join([]string{"https://www.twitch.tv", username}, "/"),
	}

	_, err = m.discord.ChannelMessageSendComplex(m.attrs.MustString("discord_channel_id", nil), &discordgo.MessageSend{
		Content: postText,
		Embed:   msgEmbed,
	})

	return errors.Wrap(err, "sending message")
}
