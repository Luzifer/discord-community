package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Luzifer/go_helpers/v2/backoff"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	twitchAPIRequestLimit   = 5
	twitchAPIRequestTimeout = 2 * time.Second
	streamScheduleEntries   = 5
)

type twitchStreamScheduleResponse struct {
	Data struct {
		Segments []struct {
			ID            string     `json:"id"`
			StartTime     *time.Time `json:"start_time"`
			EndTime       *time.Time `json:"end_time"`
			Title         string     `json:"title"`
			CanceledUntil *time.Time `json:"canceled_until"`
			Category      *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
			IsRecurring bool `json:"is_recurring"`
		} `json:"segments"`
		BroadcasterID    string `json:"broadcaster_id"`
		BroadcasterName  string `json:"broadcaster_name"`
		BroadcasterLogin string `json:"broadcaster_login"`
		Vacation         *struct {
			StartTime *time.Time `json:"start_time"`
			EndTime   *time.Time `json:"end_time"`
		} `json:"vacation"`
	} `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func init() {
	if _, err := crontab.AddFunc("*/5 * * * *", cronUpdateSchedule); err != nil {
		log.WithError(err).Fatal("Unable to add cronUpdatePresence function")
	}
}

func cronUpdateSchedule() {
	var data twitchStreamScheduleResponse
	if err := backoff.NewBackoff().WithMaxIterations(twitchAPIRequestLimit).Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), twitchAPIRequestTimeout)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.twitch.tv/helix/schedule?broadcaster_id=%s&first=%d", twitchChannelID, streamScheduleEntries), nil)
		req.Header.Set("Authorization", "Bearer "+twitchToken)
		req.Header.Set("Client-Id", twitchClientID)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "fetching schedule")
		}
		defer resp.Body.Close()

		return errors.Wrap(
			json.NewDecoder(resp.Body).Decode(&data),
			"decoding schedule response",
		)
	}); err != nil {
		log.WithError(err).Error("Unable to fetch stream schedule")
		return
	}

	msgEmbed := &discordgo.MessageEmbed{
		Color:       3066993,
		Type:        discordgo.EmbedTypeRich,
		Title:       "Fortlaufender Streamplan",
		Description: "Streams sind bis ca. 23 Uhr / Mitternacht, geplant aber man weiss ja wie das mit Plänen und Theorien so funktioniert…",
		Fields:      []*discordgo.MessageEmbedField{},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    "https://p.hub.luzifer.io/http://www.clker.com/cliparts/c/f/5/7/1194984495549320725tabella_architetto_franc_01.svg.hi.png",
			Width:  600,
			Height: 599,
		},
	}

	for _, seg := range data.Data.Segments {
		title := seg.Title
		if seg.Category != nil && seg.Category.Name != seg.Title {
			title = fmt.Sprintf("%s (%s)", seg.Title, seg.Category.Name)
		}

		msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   seg.StartTime.Format("Mon 02.01. 15:04 Uhr"),
			Value:  title,
			Inline: false,
		})
	}

	msgs, err := discord.ChannelMessages(discordAnnouncementChannel, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("Unable to fetch announcement channel messages")
		return
	}

	var msgID string
	for _, msg := range msgs {
		if len(msg.Embeds) == 0 || msg.Embeds[0].Title != msgEmbed.Title {
			continue
		}

		msgID = msg.ID
	}

	if msgID != "" {
		_, err = discord.ChannelMessageEditEmbed(discordAnnouncementChannel, msgID, msgEmbed)
	} else {
		_, err = discord.ChannelMessageSendEmbed(discordAnnouncementChannel, msgEmbed)
	}
	if err != nil {
		log.WithError(err).Error("Unable to announce streamplan")
		return
	}
}
