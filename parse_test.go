package clusterix

import (
	"testing"
	"time"

	"github.com/ftl/hamradio/callsign"
	"github.com/ftl/hamradio/locator"
	"github.com/stretchr/testify/assert"
)

func TestExtractDXMessages(t *testing.T) {
	year, month, day := time.Now().Date()
	now := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	tt := []struct {
		desc     string
		value    string
		expected []DXMessage
		invalid  bool
	}{
		{
			desc:  "dxspider",
			value: "dx de rx7k:       7154.0  rk7r         cq                             1650z kn75\a\a\r\ndx de sv1smx:    14090.0  vk9nt        tks  qso 73\\' f/h              1650z\a\a",
			expected: []DXMessage{
				{
					Call:      callsign.MustParse("rk7r"),
					Frequency: 7154000,
					Time:      now.Add(16 * time.Hour).Add(50 * time.Minute),
					Locator:   locator.MustParse("kn75"),
					Text:      "cq",
					Spotter:   "rx7k",
				},
				{
					Call:      callsign.MustParse("vk9nt"),
					Frequency: 14090000,
					Time:      now.Add(16 * time.Hour).Add(50 * time.Minute),
					Text:      "tks  qso 73\\' f/h",
					Spotter:   "sv1smx",
				},
			},
		},
		{
			desc:  "cwskimmer",
			value: "dx de dl8las-#:  14040.1  sv1rrv         20 db  18 wpm  cq            1711z",
			expected: []DXMessage{
				{
					Call:      callsign.MustParse("sv1rrv"),
					Frequency: 14040100,
					Time:      now.Add(17 * time.Hour).Add(11 * time.Minute),
					Text:      "20 db  18 wpm  cq",
					Spotter:   "dl8las-#",
				},
			},
		},
		{
			desc:  "arc6",
			value: "dx de dm5gg-#:    7003.0  4x0aaw       cw 8 db 28 wpm cq              1801z\r\ndx de sq7bfc-13: 24902.0  pj2/k5pi     tnx qso 73                     1801z\r\ndx de dm5gg-#:    7010.0  uv2iz        cw 17 db 18 wpm cq             1801z\r\ndx de dm5gg-#:   24895.0  ve3qam       cw 10 db 30 wpm cq             1801z",
			expected: []DXMessage{
				{
					Call:      callsign.MustParse("4x0aaw"),
					Frequency: 7003000,
					Time:      now.Add(18 * time.Hour).Add(01 * time.Minute),
					Text:      "cw 8 db 28 wpm cq",
					Spotter:   "dm5gg-#",
				},
				{
					Call:      callsign.MustParse("pj2/k5pi"),
					Frequency: 24902000,
					Time:      now.Add(18 * time.Hour).Add(01 * time.Minute),
					Text:      "tnx qso 73",
					Spotter:   "sq7bfc-13",
				},
				{
					Call:      callsign.MustParse("uv2iz"),
					Frequency: 7010000,
					Time:      now.Add(18 * time.Hour).Add(01 * time.Minute),
					Text:      "cw 17 db 18 wpm cq",
					Spotter:   "dm5gg-#",
				},
				{
					Call:      callsign.MustParse("ve3qam"),
					Frequency: 24895000,
					Time:      now.Add(18 * time.Hour).Add(01 * time.Minute),
					Text:      "cw 10 db 30 wpm cq",
					Spotter:   "dm5gg-#",
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := ExtractDXMessages(tc.value)
			if tc.invalid {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestParseDXTimestamp(t *testing.T) {
	now := time.Now()
	year, month, day := now.Date()
	expected := time.Date(year, month, day, 16, 50, 0, 0, time.UTC)

	actual, err := parseDXTimestamp(now, "1650")
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestTimeTruncate(t *testing.T) {
	now := time.Date(2023, time.March, 30, 19, 44, 12, 0, time.Local)
	expected := time.Date(2023, time.March, 30, 0, 0, 0, 0, time.UTC)

	actual := now.UTC().Truncate(time.Hour * 24)

	assert.Equal(t, expected, actual)
}
