package livetv

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

var extinfRe = regexp.MustCompile(`#EXTINF:\s*(-?\d+)\s*(.*)`)
var satipFreqRe = regexp.MustCompile(`[?&]freq=(\d+)`)
var satipPidsRe = regexp.MustCompile(`([?&])pids=([\d,]+)`)
var channellistRe = regexp.MustCompile(`/dvb/m3u/[^/]+\.m3u8?$`)

type M3UAttribute struct {
	ChannelNumber int
	TVGID         string
	TVGLogo       string
	GroupTitle    string
	Name          string
}

// ParseM3UChannel parses attributes from an #EXTINF line.
// Supports both attribute-rich and simple formats:
//   #EXTINF:-1 tvg-id="yle1" tvg-logo="..." group-title="Finnish",Yle TV1
//   #EXTINF:0,Yle TV1 HD
func ParseM3UChannel(line string) M3UAttribute {
	attr := M3UAttribute{}

	matches := extinfRe.FindStringSubmatch(line)
	if matches == nil {
		return attr
	}

	trailing := matches[2]

	// Check for comma separating attributes from name
	commaIdx := strings.LastIndex(trailing, ",")
	if commaIdx >= 0 {
		attr.Name = strings.TrimSpace(trailing[commaIdx+1:])
		trailing = strings.TrimSpace(trailing[:commaIdx])
	} else {
		attr.Name = strings.TrimSpace(trailing)
		return attr
	}

	// Parse key="value" pairs
	re := regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
	pairs := re.FindAllStringSubmatch(trailing, -1)
	for _, p := range pairs {
		switch p[1] {
		case "tvg-id":
			attr.TVGID = p[2]
		case "tvg-logo":
			attr.TVGLogo = p[2]
		case "group-title":
			attr.GroupTitle = p[2]
		case "tvg-chno":
			if n, err := strconv.Atoi(p[2]); err == nil {
				attr.ChannelNumber = n
			}
		}
	}

	return attr
}

// M3UEntry represents a single parsed channel entry from an M3U playlist
type M3UEntry struct {
	ChannelNumber int
	Name          string
	StreamURL     string
	LogoURL       string
	Group         string
	EPGChannelID  string
}

// ParseM3U parses an IPTV M3U playlist into channel entries
func ParseM3U(r io.Reader) ([]M3UEntry, error) {
	var entries []M3UEntry
	scanner := bufio.NewScanner(r)

	var currentAttr M3UAttribute

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || line == "#EXTM3U" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			currentAttr = ParseM3UChannel(line)
			continue
		}

		if strings.HasPrefix(line, "#EXTVLCOPT:") {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		// This is a stream URL line
		entry := M3UEntry{
			StreamURL:    line,
			ChannelNumber: currentAttr.ChannelNumber,
			Name:         currentAttr.Name,
			LogoURL:      currentAttr.TVGLogo,
			Group:        currentAttr.GroupTitle,
			EPGChannelID: currentAttr.TVGID,
		}
		entries = append(entries, entry)
		currentAttr = M3UAttribute{}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// ToDomainChannel converts a parsed M3U entry to a domain Channel entity
func (e M3UEntry) ToDomainChannel(libraryID int64) *livetv.Channel {
	return &livetv.Channel{
		LibraryID:    libraryID,
		ChannelNumber: e.ChannelNumber,
		Name:         e.Name,
		StreamURL:    e.StreamURL,
		LogoURL:      e.LogoURL,
		Group:        e.Group,
		EPGChannelID: e.EPGChannelID,
		Enabled:      true,
	}
}

// IsChannelListURL checks if a URL points to a SAT>IP channellist M3U.
func IsChannelListURL(url string) bool {
	return channellistRe.MatchString(url)
}

// SatipStreamParams holds parsed SAT>IP stream parameters.
type SatipStreamParams struct {
	Frequency int
	PIDs      string
}

// ParseSatipStreamURL extracts frequency and PIDs from a SAT>IP RTSP URL.
// Returns zero-value params if no freq/pids found.
func ParseSatipStreamURL(url string) SatipStreamParams {
	var params SatipStreamParams

	freqMatch := satipFreqRe.FindStringSubmatch(url)
	if len(freqMatch) >= 2 {
		if f, err := strconv.Atoi(freqMatch[1]); err == nil {
			params.Frequency = f
		}
	}

	pidsMatch := satipPidsRe.FindStringSubmatch(url)
	if len(pidsMatch) >= 3 {
		params.PIDs = pidsMatch[2]
	}

	return params
}

// CorrectPIDs replaces the pids= parameter in a SAT>IP stream URL with the given value.
// Returns the original URL if no pids= parameter is found.
func CorrectPIDs(streamURL string, correctPIDs string) string {
	return satipPidsRe.ReplaceAllString(streamURL, "${1}pids="+correctPIDs)
}

// FilterChannelListEntry separates the channellist meta-entry from normal channel entries.
// Returns the filtered entries, the channellist URL (if found), and any error.
func FilterChannelListEntry(entries []M3UEntry) ([]M3UEntry, string, error) {
	var filtered []M3UEntry
	channellistURL := ""

	for _, entry := range entries {
		if IsChannelListURL(entry.StreamURL) {
			channellistURL = entry.StreamURL
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered, channellistURL, nil
}
