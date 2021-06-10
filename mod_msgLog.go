package main

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

func init() {
	discordHandlers = append(discordHandlers, handleMessageDeleteLog)
}

func handleMessageDeleteLog(s *discordgo.Session, m *discordgo.MessageDelete) {
	if m.Author.ID == s.State.User.ID {
		// Bot was the author, do not spam
		return
	}

	// FIXME: Do something useful with this
	log.WithField("msg", m).Info("Message deleted")
}
