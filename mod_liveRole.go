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
}

func (m *modLiveRole) Initialize(crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error {
	m.attrs = attrs
	m.discord = discord

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

func (m modLiveRole) addLiveStreamerRole(guildID, userID string, presentRoles []string) error {
	// @attr role_streamers_live required string "" Role ID to assign to live streamers
	roleID := m.attrs.MustString("role_streamers_live", nil)
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

	logger := log.WithFields(log.Fields{
		"user": p.User.ID,
	})

	member, err := d.GuildMember(p.GuildID, p.User.ID)
	if err != nil {
		logger.WithError(err).Error("Unable to fetch member status for user")
		return
	}

	// @attr role_streamers optional string "" Only take members with this role into account
	roleStreamer := m.attrs.MustString("role_streamers", ptrStringEmpty)
	if roleStreamer != "" && !str.StringInSlice(roleStreamer, member.Roles) {
		// User is not part of the streamer role
		return
	}

	var exitFunc func(string, string, []string) error = m.removeLiveStreamerRole
	defer func() {
		if exitFunc != nil {
			if err := exitFunc(p.GuildID, p.User.ID, member.Roles); err != nil {
				logger.WithError(err).Error("Unable to update live-streamer-role")
			}
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
		return
	}

	u, err := url.Parse(activity.URL)
	if err != nil {
		logger.WithError(err).WithField("url", activity.URL).Warning("Unable to parse activity URL")
		exitFunc = m.removeLiveStreamerRole
		return
	}

	if u.Host != "www.twitch.tv" {
		logger.WithError(err).WithField("url", activity.URL).Warning("Activity is not on Twitch")
		exitFunc = m.removeLiveStreamerRole
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
		return
	}

	if len(streams.Data) > 0 {
		exitFunc = m.addLiveStreamerRole
	}
}

func (m modLiveRole) removeLiveStreamerRole(guildID, userID string, presentRoles []string) error {
	roleID := m.attrs.MustString("role_streamers_live", nil)
	if !str.StringInSlice(roleID, presentRoles) {
		// Not there: fine!
		return nil
	}

	return errors.Wrap(
		m.discord.GuildMemberRoleRemove(guildID, userID, roleID),
		"adding role",
	)
}
