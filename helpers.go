package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	ptrBoolFalse   = ptrBool(false)
	ptrInt64Zero   = ptrInt64(0)
	ptrStringEmpty = ptrString("")
)

func ptrBool(v bool) *bool                       { return &v }
func ptrDuration(v time.Duration) *time.Duration { return &v }
func ptrInt64(v int64) *int64                    { return &v }
func ptrString(v string) *string                 { return &v }
func ptrTime(v time.Time) *time.Time             { return &v }

//nolint:gocognit,gocyclo // This function compares two structs and needs the complexity
func isDiscordMessageEmbedEqual(a, b *discordgo.MessageEmbed) bool {
	if a == nil || b == nil {
		// If one of them is nil, don't do the in-depth analysis
		return a == b
	}

	checks := [][2]interface{}{
		// Base-Struct
		{a.Type, b.Type},
		{a.URL, b.URL},
		{a.Title, b.Title},
		{a.Description, b.Description},
		// We ignore the timestamp here as it would represent the post time of the new embed
		{a.Color, b.Color},
	}

	if a.Footer == nil && b.Footer != nil || a.Footer != nil && b.Footer == nil {
		return false
	}
	if a.Image == nil && b.Image != nil || a.Image != nil && b.Image == nil {
		return false
	}
	if a.Thumbnail == nil && b.Thumbnail != nil || a.Thumbnail != nil && b.Thumbnail == nil {
		return false
	}
	if a.Video == nil && b.Video != nil || a.Video != nil && b.Video == nil {
		return false
	}
	if a.Provider == nil && b.Provider != nil || a.Provider != nil && b.Provider == nil {
		return false
	}
	if a.Author == nil && b.Author != nil || a.Author != nil && b.Author == nil {
		return false
	}

	if a.Footer != nil {
		checks = append(checks, [][2]interface{}{
			{a.Footer.IconURL, b.Footer.IconURL},
			{a.Footer.Text, b.Footer.Text},
		}...)
	}

	if a.Image != nil {
		checks = append(checks, [][2]interface{}{
			{a.Image.URL, b.Image.URL},
			{a.Image.Width, b.Image.Width},
			{a.Image.Height, b.Image.Height},
		}...)
	}

	if a.Thumbnail != nil {
		checks = append(checks, [][2]interface{}{
			{a.Thumbnail.URL, b.Thumbnail.URL},
			{a.Thumbnail.Width, b.Thumbnail.Width},
			{a.Thumbnail.Height, b.Thumbnail.Height},
		}...)
	}

	if a.Video != nil {
		checks = append(checks, [][2]interface{}{
			{a.Video.URL, b.Video.URL},
			{a.Video.Width, b.Video.Width},
			{a.Video.Height, b.Video.Height},
		}...)
	}

	if a.Provider != nil {
		checks = append(checks, [][2]interface{}{
			{a.Provider.URL, b.Provider.URL},
			{a.Provider.Name, b.Provider.Name},
		}...)
	}

	if a.Author != nil {
		checks = append(checks, [][2]interface{}{
			{a.Author.URL, b.Author.URL},
			{a.Author.Name, b.Author.Name},
			{a.Author.IconURL, b.Author.IconURL},
		}...)
	}

	if len(a.Fields) != len(b.Fields) {
		return false
	}

	for i := range a.Fields {
		checks = append(checks, [][2]interface{}{
			{a.Fields[i].Name, b.Fields[i].Name},
			{a.Fields[i].Value, b.Fields[i].Value},
			{a.Fields[i].Inline, b.Fields[i].Inline},
		}...)
	}

	for _, p := range checks {
		if p[0] != p[1] {
			return false
		}
	}

	return true
}
