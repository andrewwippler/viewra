package livetv

import (
	"encoding/xml"
	"io"
	"time"
	"strings"
	"strconv"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

// XMLTV XML structures for parsing
type xmltv struct {
	XMLName    xml.Name          `xml:"tv"`
	Channels   []xmltvChannel    `xml:"channel"`
	Programmes []xmltvProgramme  `xml:"programme"`
}

type xmltvChannel struct {
	ID      string `xml:"id,attr"`
	Display []struct {
		Text string `xml:",chardata"`
		Lang string `xml:"lang,attr"`
	} `xml:"display-name"`
}

type xmltvProgramme struct {
	Start   string `xml:"start,attr"`
	Stop    string `xml:"stop,attr"`
	Channel string `xml:"channel,attr"`

	Title      string `xml:"title"`
	SubTitle   string `xml:"sub-title"`
	Desc       string `xml:"desc"`
	Category   string `xml:"category"`
	EpisodeNum string `xml:"episode-num"`
	Rating     struct {
		Value string `xml:"value"`
	} `xml:"rating"`
	StarRating struct {
		Value string `xml:"value"`
	} `xml:"star-rating"`
}

// xmltvTime formats: "20260613040200 +0000" or "20260613040200 +0300"
func parseXMLTVTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) < 1 {
		return time.Time{}, nil
	}

	t := parts[0]
	// Format: YYYYMMDDHHMMSS
	if len(t) != 14 {
		return time.Time{}, nil
	}

	year, _ := strconv.Atoi(t[0:4])
	month, _ := strconv.Atoi(t[4:6])
	day, _ := strconv.Atoi(t[6:8])
	hour, _ := strconv.Atoi(t[8:10])
	min, _ := strconv.Atoi(t[10:12])
	sec, _ := strconv.Atoi(t[12:14])

	loc := time.UTC
	if len(parts) >= 2 {
		offset := parts[1]
		if len(offset) == 5 && (offset[0] == '+' || offset[0] == '-') {
			oh, _ := strconv.Atoi(offset[1:3])
			om, _ := strconv.Atoi(offset[3:5])
			if offset[0] == '-' {
				oh = -oh
				om = -om
			}
			loc = time.FixedZone(offset, oh*3600+om*60)
		}
	}

	return time.Date(year, time.Month(month), day, hour, min, sec, 0, loc), nil
}

// XMLTVParseResult contains the parsed EPG data
type XMLTVParseResult struct {
	ChannelMap map[string]string // EPG channel ID -> channel name
	Programs   []XMLTVProgram
}

// XMLTVProgram represents a parsed EPG program entry
type XMLTVProgram struct {
	ChannelID  string
	StartTime  time.Time
	EndTime    time.Time
	Title      string
	SubTitle   string
	Desc       string
	Category   string
	EpisodeNum string
}

// ParseXMLTV parses an XMLTV EPG feed into programs
func ParseXMLTV(r io.Reader) (*XMLTVParseResult, error) {
	var tv xmltv
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&tv); err != nil {
		return nil, err
	}

	result := &XMLTVParseResult{
		ChannelMap: make(map[string]string),
	}

	for _, ch := range tv.Channels {
		name := ch.ID
		if len(ch.Display) > 0 {
			name = ch.Display[0].Text
		}
		result.ChannelMap[ch.ID] = name
	}

	for _, p := range tv.Programmes {
		start, err := parseXMLTVTime(p.Start)
		if err != nil {
			continue
		}
		end, err := parseXMLTVTime(p.Stop)
		if err != nil {
			continue
		}

		prog := XMLTVProgram{
			ChannelID:  p.Channel,
			StartTime:  start,
			EndTime:    end,
			Title:      p.Title,
			SubTitle:   p.SubTitle,
			Desc:       p.Desc,
			Category:   p.Category,
			EpisodeNum: p.EpisodeNum,
		}
		result.Programs = append(result.Programs, prog)
	}

	return result, nil
}

// ToDomainPrograms converts parsed XMLTV programs to domain Program entities
// using a mapping from epgChannelID -> channelID in the database
func ToDomainPrograms(programs []XMLTVProgram, epgToChannelID map[string]int64) []*livetv.Program {
	var result []*livetv.Program

	for _, p := range programs {
		channelID, ok := epgToChannelID[p.ChannelID]
		if !ok {
			continue
		}

		isNew := strings.Contains(strings.ToLower(p.Category), "new") ||
			strings.Contains(p.Title, "(U)") ||
			strings.Contains(p.Desc, "(U)")

		isMovie := strings.Contains(strings.ToLower(p.Category), "movie") ||
			strings.Contains(strings.ToLower(p.Category), "film")

		prog := &livetv.Program{
			ChannelID:   channelID,
			StartTime:   p.StartTime,
			EndTime:     p.EndTime,
			Title:       p.Title,
			SubTitle:    p.SubTitle,
			Description: p.Desc,
			Category:    p.Category,
			IsNew:       isNew,
			IsMovie:     isMovie,
		}

		result = append(result, prog)
	}

	return result
}
