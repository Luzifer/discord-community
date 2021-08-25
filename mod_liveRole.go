package main

import (
	"context"
	"net/url"
	"strings"

	"github.com/Luzifer/go_helpers/v2/str"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

/*
 * @module liverole
 * @module_desc Adds live-role to certain group of users if they are streaming on Twitch
 */

func init() {
	RegisterModule("liverole", func() module { return &modLiveRole{} })
}

type modLiveRole struct {
	attrs   moduleAttributeStore
	discord *discordgo.Session
	id      string
}

func (m modLiveRole) ID() string { return m.id }

func (m *modLiveRole) Initialize(id string, crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord
	m.id = id

	if err := attrs.Expect(
		"role_streamers_live",
		"twitch_client_id",
		"twitch_client_secret",
	); err != nil {
		return errors.Wrap(err, "validating attributes")
	}

	discord.AddHandler(m.handlePresenceUpdate)

	return nil
}

func (m modLiveRole) Setup() error { return nil }

func (m modLiveRole) addLiveStreamerRole(guildID, userID string, presentRoles []string) error {
	// @attr role_streamers_live required string "" Role ID to assign to live streamers (make sure the bot [can assign](https://support.discord.com/hc/en-us/articles/214836687-Role-Management-101) this role)
	roleID := m.attrs.MustString("role_streamers_live", nil)
	if roleID == "" {
		return errors.New("empty live-role-id")
	}
	if str.StringInSlice(roleID, presentRoles) {
		// Already there fine!
		return nil
	}

	return errors.Wrap(
		m.discord.GuildMemberRoleAdd(guildID, userID, roleID),
		"adding role",
	)
}

func (m modLiveRole) handlePresenceUpdate(d *discordgo.Session, p *discordgo.PresenceUpdate) {
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

	// @attr role_streamers optional string "" Only take members with this role ID into account
	roleStreamer := m.attrs.MustString("role_streamers", ptrStringEmpty)
	if roleStreamer != "" && !str.StringInSlice(roleStreamer, member.Roles) {
		// User is not part of the streamer role
		return
	}

	var exitFunc func(string, string, []string) error
	defer func() {
		if exitFunc != nil {
			if err := exitFunc(p.GuildID, p.User.ID, member.Roles); err != nil {
				logger.WithError(err).Error("Unable to update live-streamer-role")
			}
			logger.Debug("Updated live-streamer-role")
		}
	}()

	var activity *discordgo.Activity

	for _, a := range p.Activities {
		if a.Type == discordgo.ActivityTypeStreaming {
			activity = a
			break
		}
	}

	if activity == nil {
		// No streaming activity: Remove role
		exitFunc = m.removeLiveStreamerRole
		logger = logger.WithFields(log.Fields{"action": "remove", "reason": "no activity"})
		return
	}

	u, err := url.Parse(activity.URL)
	if err != nil {
		logger.WithError(err).WithField("url", activity.URL).Warning("Unable to parse activity URL")
		exitFunc = m.removeLiveStreamerRole
		logger = logger.WithFields(log.Fields{"action": "remove", "reason": "broken activity URL"})
		return
	}

	if u.Host != "www.twitch.tv" {
		logger.WithError(err).WithField("url", activity.URL).Warning("Activity is not on Twitch")
		exitFunc = m.removeLiveStreamerRole
		logger = logger.WithFields(log.Fields{"action": "remove", "reason": "activity not on twitch"})
		return
	}

	twitch := newTwitchAdapter(
		// @attr twitch_client_id required string "" Twitch client ID the token was issued for
		m.attrs.MustString("twitch_client_id", nil),
		// @attr twitch_client_secret required string "" Secret for the Twitch app identified with twitch_client_id
		m.attrs.MustString("twitch_client_secret", nil),
		"", // No User-Token used
	)

	streams, err := twitch.GetStreamsForUser(context.Background(), strings.TrimLeft(u.Path, "/"))
	if err != nil {
		logger.WithError(err).WithField("user", strings.TrimLeft(u.Path, "/")).Warning("Unable to fetch streams for user")
		exitFunc = m.removeLiveStreamerRole
		logger = logger.WithFields(log.Fields{"action": "remove", "reason": "error in getting streams"})
		return
	}

	if len(streams.Data) > 0 {
		exitFunc = m.addLiveStreamerRole
		logger = logger.WithFields(log.Fields{"action": "add", "reason": "stream found"})
	}
}

func (m modLiveRole) removeLiveStreamerRole(guildID, userID string, presentRoles []string) error {
	roleID := m.attrs.MustString("role_streamers_live", nil)
	if roleID == "" {
		return errors.New("empty live-role-id")
	}
	if !str.StringInSlice(roleID, presentRoles) {
		// Not there: fine!
		return nil
	}

	return errors.Wrap(
		m.discord.GuildMemberRoleRemove(guildID, userID, roleID),
		"adding role",
	)
}
