package clusterix

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ftl/hamradio/callsign"
	"github.com/ftl/hamradio/locator"
)

type DXMessage struct {
	Call      callsign.Callsign
	Frequency float64
	Time      time.Time
	Locator   locator.Locator
	Text      string
	Spotter   string
}

var dxMessageExpression = regexp.MustCompile(`dx de (.+):\s+([0-9]+\.[0-9]+)\s+([0-9a-z/]+)\s+(.+)\s+([0-9]{4})z( [a-z]{2}[0-9]{2})?`)

func ExtractDXMessages(s string) ([]DXMessage, error) {
	matches := dxMessageExpression.FindAllStringSubmatch(s, -1)
	result := make([]DXMessage, 0, len(matches))

	for _, match := range matches {
		if len(match) != 7 {
			return nil, fmt.Errorf("invalid match count: %d %v", len(match), matches)
		}
		call, err := callsign.Parse(match[3])
		if err != nil {
			continue
		}
		frequency, err := parseDXFrequency(match[2])
		if err != nil {
			continue
		}
		timestamp, err := parseDXTimestamp(time.Now(), match[5])
		if err != nil {
			continue
		}
		loc, err := parseDXLocator(match[6])
		if err != nil {
			loc = locator.Locator{}
		}

		result = append(result, DXMessage{
			Call:      call,
			Frequency: frequency,
			Time:      timestamp,
			Locator:   loc,
			Text:      strings.TrimSpace(match[4]),
			Spotter:   match[1],
		})
	}

	return result, nil
}

func parseDXFrequency(s string) (float64, error) {
	kHz, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return kHz * 1000, nil
}

func parseDXTimestamp(now time.Time, timestamp string) (time.Time, error) {
	if len(timestamp) != 4 {
		return time.Time{}, fmt.Errorf("invalid timestamp: %s", timestamp)
	}
	hours, err := strconv.Atoi(timestamp[:2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %v", err)
	}
	minutes, err := strconv.Atoi(timestamp[2:])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %v", err)
	}
	today := now.UTC().Truncate(time.Hour * 24)

	return today.Add(time.Duration(hours) * time.Hour).Add(time.Duration(minutes) * time.Minute), nil
}

func parseDXLocator(s string) (locator.Locator, error) {
	return locator.Parse(strings.TrimSpace(s))
}
