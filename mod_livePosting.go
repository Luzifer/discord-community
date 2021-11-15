package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	livePostingPreviewHeight          = 720
	livePostingPreviewWidth           = 1280
	livePostingTwitchColor            = 0x6441a5
)

func init() {
	RegisterModule("liveposting", func() module { return &modLivePosting{} })
}

type modLivePosting struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string

	lock sync.Mutex
}

func (m *modLivePosting) ID() string { return m.id }

func (m *modLivePosting) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

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

func (m *modLivePosting) Setup() error { return nil }

func (m *modLivePosting) cronFetchChannelStatus() {
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

	log.WithField("entries", len(usernames)).Trace("Fetching streams for users (cron)")

	if err = m.fetchAndPostForUsername(usernames...); err != nil {
		log.WithError(err).Error("Unable to post status for users")
	}
}

func (m *modLivePosting) fetchAndPostForUsername(usernames ...string) error {
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

	log.WithFields(log.Fields{
		"streams": len(streams.Data),
		"users":   len(users.Data),
	}).Trace("Found active streams from users")

	// @attr stream_freshness optional duration "5m" How long after stream start to post shoutout
	streamFreshness := m.attrs.MustDuration("stream_freshness", ptrDuration(livePostingDefaultStreamFreshness))

	for _, stream := range streams.Data {
		for _, user := range users.Data {
			if user.ID != stream.UserID {
				continue
			}

			isFresh := time.Since(stream.StartedAt) <= streamFreshness

			log.WithFields(log.Fields{
				"isFresh":    isFresh,
				"started_at": stream.StartedAt,
				"user":       user.DisplayName,
			}).Trace("Found user / stream combination")

			if !isFresh {
				// Stream is too old, don't annoounce
				continue
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

func (m *modLivePosting) handlePresenceUpdate(d *discordgo.Session, p *discordgo.PresenceUpdate) {
	if p.User == nil {
		// The frick? Non-user presence?
		return
	}

	if p.GuildID != config.GuildID {
		// Bot is in multiple guilds, we don't have a config for this one
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

	// @attr whitelisted_role optional string "" Only post for members of this role ID
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

//nolint:funlen // Makes no sense to split just for 2 lines
func (m *modLivePosting) sendLivePost(username, displayName, title, game, previewImage, profileImage string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	logger := log.WithFields(log.Fields{
		"user": username,
		"game": game,
	})

	// @attr post_text required string "" Message to post to channel use `${displayname}` and `${username}` as placeholders
	postTemplateDefault := m.attrs.MustString("post_text", nil)

	postText := strings.NewReplacer(
		"${displayname}", displayName,
		"${username}", username,
	).Replace(
		// @attr post_text_{username} optional string "" Override the default `post_text` with this one (e.g. `post_text_luziferus: "${displayName} is now live"`)
		m.attrs.MustString(fmt.Sprintf("post_text_%s", strings.ToLower(username)), &postTemplateDefault),
	)

	// @attr discord_channel_id required string "" ID of the Discord channel to post the message to
	channelID := m.attrs.MustString("discord_channel_id", nil)

	msgs, err := m.discord.ChannelMessages(channelID, livePostingNumberOfMessagesToLoad, "", "", "")
	if err != nil {
		return errors.Wrap(err, "fetching previous messages")
	}

	ignoreTime := m.attrs.MustDuration("stream_freshness", ptrDuration(livePostingDefaultStreamFreshness))
	for _, msg := range msgs {
		mt, err := msg.Timestamp.Parse()
		if err != nil {
			return errors.Wrap(err, "parsing message timestamp")
		}

		if msg.Content != postText {
			// Post is for another channel / is another message
			continue
		}

		if time.Since(mt) < ignoreTime {
			// Message is still fresh
			logger.Debug("Not creating live-post, it's already there")
			return nil
		}

		// @attr remove_old optional bool "false" If set to `true` older message with same content will be deleted
		if !m.attrs.MustBool("remove_old", ptrBoolFalse) {
			// We're not allowed to purge the old message
			continue
		}

		// Purge the old message
		if err = m.discord.ChannelMessageDelete(channelID, msg.ID); err != nil {
			return errors.Wrap(err, "deleting old message")
		}
	}

	// Discord caches the images and the URLs do not change every time
	// so we force Discord to load a new image every time
	previewImageURL, err := url.Parse(
		strings.NewReplacer(
			"{width}", strconv.Itoa(livePostingPreviewWidth),
			"{height}", strconv.Itoa(livePostingPreviewHeight),
		).Replace(previewImage),
	)
	if err != nil {
		return errors.Wrap(err, "parsing stream preview URL")
	}

	previewImageQuery := previewImageURL.Query()
	previewImageQuery.Add("_discordNoCache", time.Now().Format(time.RFC3339))
	previewImageURL.RawQuery = previewImageQuery.Encode()

	// @attr preserve_proxy optional string "" URL prefix of a Luzifer/preserve proxy to cache stream preview for longer
	if proxy, err := url.Parse(m.attrs.MustString("preserve_proxy", ptrStringEmpty)); err == nil && proxy.String() != "" {
		// Discord screws up the plain-text URL format, so we need to use the b64-format
		proxy.Path = "/b64:" + base64.URLEncoding.EncodeToString([]byte(previewImageURL.String()))
		previewImageURL = proxy
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
			URL:    previewImageURL.String(),
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

	logger.Debug("Creating live-post")

	msg, err := m.discord.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: postText,
		Embed:   msgEmbed,
	})
	if err != nil {
		return errors.Wrap(err, "sending message")
	}

	// @attr auto_publish optional bool "false" Automatically publish (crosspost) the message to followers of the channel
	if m.attrs.MustBool("auto_publish", ptrBoolFalse) {
		logger.Debug("Auto-Publishing live-post")
		if _, err = m.discord.ChannelMessageCrosspost(channelID, msg.ID); err != nil {
			return errors.Wrap(err, "publishing message")
		}
	}

	return nil
}
