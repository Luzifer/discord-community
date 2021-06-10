package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	if _, err := crontab.AddFunc("* * * * *", cronUpdatePresence); err != nil {
		log.WithError(err).Fatal("Unable to add cronUpdatePresence function")
	}
}

func cronUpdatePresence() {
	var nextStream *time.Time = nil

	// FIXME: Get next stream status

	status := "mit Seelen"
	if nextStream != nil {
		status = fmt.Sprintf("in: %s", durationToHumanReadable(time.Since(*nextStream)))
	}

	if err := discord.UpdateGameStatus(0, status); err != nil {
		log.WithError(err).Error("Unable to update status")
	}

	log.Debug("Updated presence")
}

func durationToHumanReadable(d time.Duration) string {
	var elements []string

	d = time.Duration(math.Abs(float64(d)))
	for div, req := range map[time.Duration]bool{
		time.Hour * 24: false,
		time.Hour:      true,
		time.Minute:    true,
	} {
		if d < div && !req {
			continue
		}

		pt := d / div
		d -= pt * div
		elements = append(elements, fmt.Sprintf("%.2d", pt))
	}

	return strings.Join(elements, ":")
}
