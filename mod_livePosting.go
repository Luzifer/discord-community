package main

import (
	"context"
	"net/url"
	"strconv"
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

const (
	livePostingDefaultStreamFreshness = 5 * time.Minute
	livePostingDiscordProfileHeight   = 300
	livePostingDiscordProfileWidth    = 300
	livePostingNumberOfMessagesToLoad = 100
	livePostingPreviewHeight          = 180
	livePostingPreviewWidth           = 320
	livePostingTwitchColor            = 0x6441a5
)

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
		"twitch_client_id",
		"twitch_client_secret",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	// @attr disable_presence optional bool "false" Disable posting live-postings for discord presence changes
	if !attrs.MustBool("disable_presence", ptrBoolFalse) {
		discord.AddHandler(m.handlePresenceUpdate)
	}

	// @attr cron optional string "*/5 * * * *" Fetch live status of `poll_usernames` (set to empty string to disable): keep this below `stream_freshness` or you might miss streams
	if cronDirective := attrs.MustString("cron", ptrString("*/5 * * * *")); cronDirective != "" {
		if _, err := crontab.AddFunc(cronDirective, m.cronFetchChannelStatus); err != nil {
			return errors.Wrap(err, "adding cron function")
		}
	}

	return nil
}

func (m modLivePosting) cronFetchChannelStatus() {
	// @attr poll_usernames optional []string "[]" Check these usernames for active streams when executing the `cron` (at most 100 users can be checked)
	usernames, err := m.attrs.StringSlice("poll_usernames")
	switch err {
	case nil:
		// We got a list of users
	case errValueNotSet:
		// There is no list of users
		return
	default:
		log.WithError(err).Error("Unable to get poll_usernames list")
		return
	}

	if err = m.fetchAndPostForUsername(usernames...); err != nil {
		log.WithError(err).Error("Unable to post status for users")
	}
}

func (m modLivePosting) fetchAndPostForUsername(usernames ...string) error {
	twitch := newTwitchAdapter(
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		m.attrs.MustString("twitch_client_id", nil),
		// @attr twitch_client_secret required string "" Secret for the Twitch app identified with twitch_client_id
		m.attrs.MustString("twitch_client_secret", nil),
		"", // No User-Token used
	)

	users, err := twitch.GetUserByUsername(context.Background(), usernames...)
	if err != nil {
		return errors.Wrap(err, "fetching twitch user details")
	}

	streams, err := twitch.GetStreamsForUser(context.Background(), usernames...)
	if err != nil {
		return errors.Wrap(err, "fetching streams for user")
	}

	for _, stream := range streams.Data {
		for _, user := range users.Data {
			if user.ID != stream.ID {
				continue
			}

			// @attr stream_freshness optional duration "5m" How long after stream start to post shoutout
			ignoreTime := m.attrs.MustDuration("stream_freshness", ptrDuration(livePostingDefaultStreamFreshness))
			if stream.StartedAt.Add(ignoreTime).Before(time.Now()) {
				// Stream is too old, don't annoounce
				return nil
			}

			if err = m.sendLivePost(
				user.Login,
				user.DisplayName,
				stream.Title,
				stream.GameName,
				stream.ThumbnailURL,
				user.ProfileImageURL,
			); err != nil {
				return errors.Wrap(err, "sending post")
			}
		}
	}

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

	if err = m.fetchAndPostForUsername(twitchUsername); err != nil {
		logger.WithError(err).WithField("url", activity.URL).Error("Unable to fetch info / post live posting")
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
	msgs, err := m.discord.ChannelMessages(m.attrs.MustString("discord_channel_id", nil), livePostingNumberOfMessagesToLoad, "", "", "")
	if err != nil {
		return errors.Wrap(err, "fetching previous messages")
	}

	ignoreTime := m.attrs.MustDuration("stream_freshness", ptrDuration(livePostingDefaultStreamFreshness))
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
		Color: livePostingTwitchColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Game", Value: game},
		},
		Image: &discordgo.MessageEmbedImage{
			URL:    strings.NewReplacer("{width}", strconv.Itoa(livePostingPreviewWidth), "{height}", strconv.Itoa(livePostingPreviewHeight)).Replace(previewImage),
			Width:  livePostingPreviewWidth,
			Height: livePostingPreviewHeight,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    profileImage,
			Width:  livePostingDiscordProfileWidth,
			Height: livePostingDiscordProfileHeight,
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
