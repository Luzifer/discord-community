package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	streamSchedulePastTime  = 15 * time.Minute
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
	if _, err := crontab.AddFunc("*/10 * * * *", cronUpdateSchedule); err != nil {
		log.WithError(err).Fatal("Unable to add cronUpdatePresence function")
	}
}

func cronUpdateSchedule() {
	var data twitchStreamScheduleResponse
	if err := backoff.NewBackoff().WithMaxIterations(twitchAPIRequestLimit).Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), twitchAPIRequestTimeout)
		defer cancel()

		u, _ := url.Parse("https://api.twitch.tv/helix/schedule")
		params := make(url.Values)
		params.Set("broadcaster_id", twitchChannelID)
		params.Set("start_time", time.Now().Add(-streamSchedulePastTime).Format(time.RFC3339))
		u.RawQuery = params.Encode()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
		Description: "Streams sind bis ca. 23 Uhr / Mitternacht, geplant aber man weiss ja wie das mit Plänen und Theorien so funktioniert…",
		Fields:      []*discordgo.MessageEmbedField{},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    "https://p.hub.luzifer.io/http://www.clker.com/cliparts/c/f/5/7/1194984495549320725tabella_architetto_franc_01.svg.hi.png",
			Width:  600,
			Height: 599,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Title:     "Fortlaufender Streamplan",
		Type:      discordgo.EmbedTypeRich,
	}

	for _, seg := range data.Data.Segments {
		title := seg.Title
		if seg.Category != nil && seg.Category.Name != seg.Title {
			title = fmt.Sprintf("%s (%s)", seg.Title, seg.Category.Name)
		}

		if seg.StartTime == nil || seg.CanceledUntil != nil {
			continue
		}

		msgEmbed.Fields = append(msgEmbed.Fields, &discordgo.MessageEmbedField{
			Name:   formatGermanShort(*seg.StartTime),
			Value:  title,
			Inline: false,
		})

		if len(msgEmbed.Fields) == streamScheduleEntries {
			break
		}
	}

	msgs, err := discord.ChannelMessages(discordAnnouncementChannel, 100, "", "", "")
	if err != nil {
		log.WithError(err).Error("Unable to fetch announcement channel messages")
		return
	}

	var managedMsg *discordgo.Message
	for _, msg := range msgs {
		if len(msg.Embeds) == 0 || msg.Embeds[0].Title != msgEmbed.Title {
			continue
		}

		managedMsg = msg
	}

	if managedMsg != nil {
		oldEmbed := managedMsg.Embeds[0]

		if !embedNeedsUpdate(oldEmbed, msgEmbed) {
			log.Debug("Stream Schedule is up-to-date")
			return
		}

		_, err = discord.ChannelMessageEditEmbed(discordAnnouncementChannel, managedMsg.ID, msgEmbed)
	} else {
		_, err = discord.ChannelMessageSendEmbed(discordAnnouncementChannel, msgEmbed)
	}
	if err != nil {
		log.WithError(err).Error("Unable to announce streamplan")
		return
	}

	log.Info("Updated Stream Schedule")
}

func formatGermanShort(t time.Time) string {
	wd := map[time.Weekday]string{
		time.Monday:    "Mo.",
		time.Tuesday:   "Di.",
		time.Wednesday: "Mi.",
		time.Thursday:  "Do.",
		time.Friday:    "Fr.",
		time.Saturday:  "Sa.",
		time.Sunday:    "So.",
	}[t.Weekday()]

	tz, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.WithError(err).Fatal("Unable to load timezone Europe/Berlin")
	}

	return strings.Join([]string{wd, t.In(tz).Format("02.01. 15:04"), "Uhr"}, " ")
}

func embedNeedsUpdate(o, n *discordgo.MessageEmbed) bool {
	if o.Title != n.Title {
		return true
	}

	if o.Description != n.Description {
		return true
	}

	if o.Thumbnail != nil && n.Thumbnail == nil || o.Thumbnail == nil && n.Thumbnail != nil {
		return true
	}

	if o.Thumbnail != nil && o.Thumbnail.URL != n.Thumbnail.URL {
		return true
	}

	if len(o.Fields) != len(n.Fields) {
		return true
	}

	for i := range o.Fields {
		if o.Fields[i].Name != n.Fields[i].Name {
			return true
		}

		if o.Fields[i].Value != n.Fields[i].Value {
			return true
		}

		if o.Fields[i].Inline != n.Fields[i].Inline {
			return true
		}
	}

	return false
}
