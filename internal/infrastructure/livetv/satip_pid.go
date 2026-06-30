package livetv

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
)

// CorrectSATIPPIDs detects a SAT>IP channellist URL in the entries (or from
// stored config), fetches the authoritative channellist, and corrects the
// PIDs on matching channels. The channellist meta-entry is removed from the
// returned slice. If no channellist is found, entries are returned unchanged.
func CorrectSATIPPIDs(
	ctx context.Context,
	entries []M3UEntry,
	lib *library.Library,
	libraryRepo library.Repository,
	channelRepo livetv.ChannelRepository,
	httpClient *http.Client,
) ([]M3UEntry, error) {
	// 1. Detect channellist URL from entries or stored config
	filtered, channellistURL, err := FilterChannelListEntry(entries)
	if err != nil {
		return entries, err
	}

	if channellistURL == "" && lib.MonitoringConfig != nil && lib.MonitoringConfig.SatipChannelListURL != "" {
		channellistURL = lib.MonitoringConfig.SatipChannelListURL
	}

	if channellistURL == "" {
		return entries, nil
	}

	// 2. Persist the channellist URL if newly detected
	if lib.MonitoringConfig == nil {
		lib.MonitoringConfig = &library.MonitoringConfig{}
	}
	if lib.MonitoringConfig.SatipChannelListURL != channellistURL {
		lib.MonitoringConfig.SatipChannelListURL = channellistURL
		if err := libraryRepo.UpdateMonitoring(ctx, lib.ID, lib.MonitoringEnabled, lib.MonitoringConfig); err != nil {
			return entries, fmt.Errorf("failed to persist channellist URL: %w", err)
		}
	}

	// 3. Fetch and parse channellist
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, channellistURL, nil)
	if err != nil {
		return filtered, fmt.Errorf("failed to create channellist request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return filtered, fmt.Errorf("failed to fetch channellist: %w", err)
	}
	defer resp.Body.Close()

	channellistEntries, err := ParseM3U(resp.Body)
	if err != nil {
		return filtered, fmt.Errorf("failed to parse channellist: %w", err)
	}

	// 4. Build frequency -> PID map from authoritative channellist
	freqToPIDs := make(map[int]string)
	for _, ce := range channellistEntries {
		params := ParseSatipStreamURL(ce.StreamURL)
		if params.Frequency > 0 && params.PIDs != "" {
			freqToPIDs[params.Frequency] = params.PIDs
		}
	}

	if len(freqToPIDs) == 0 {
		return filtered, nil
	}

	// 5. Correct PIDs on matching entries
	for i, entry := range filtered {
		params := ParseSatipStreamURL(entry.StreamURL)
		if params.Frequency == 0 || params.PIDs == "" {
			continue
		}

		correctPIDs, ok := freqToPIDs[params.Frequency]
		if !ok {
			continue
		}

		if params.PIDs == correctPIDs {
			continue
		}

		filtered[i].StreamURL = CorrectPIDs(entry.StreamURL, correctPIDs)
	}

	return filtered, nil
}

// CorrectChannelPIDsFromList uses a pre-existing channel list (frequency→PID map)
// to correct the StreamURL on an individual channel. Returns true if corrected.
func CorrectChannelPIDsFromList(ch *livetv.Channel, freqToPIDs map[int]string) bool {
	params := ParseSatipStreamURL(ch.StreamURL)
	if params.Frequency == 0 || params.PIDs == "" {
		return false
	}

	correctPIDs, ok := freqToPIDs[params.Frequency]
	if !ok || params.PIDs == correctPIDs {
		return false
	}

	ch.StreamURL = CorrectPIDs(ch.StreamURL, correctPIDs)
	return true
}

// FetchSatipChannelList fetches and parses a SAT>IP channellist, returning
// a frequency → PID map.
func FetchSatipChannelList(ctx context.Context, channellistURL string, httpClient *http.Client) (map[int]string, error) {
	parsedURL, err := url.Parse(channellistURL)
	if err != nil {
		return nil, fmt.Errorf("invalid channellist URL: %w", err)
	}

	// Resolve relative paths
	if !parsedURL.IsAbs() {
		return nil, fmt.Errorf("channellist URL must be absolute")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, channellistURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channellist: %w", err)
	}
	defer resp.Body.Close()

	entries, err := ParseM3U(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse channellist: %w", err)
	}

	freqToPIDs := make(map[int]string)
	for _, entry := range entries {
		params := ParseSatipStreamURL(entry.StreamURL)
		if params.Frequency > 0 && params.PIDs != "" {
			// If multiple entries share a frequency, prefer the one with more PIDs
			if existing, ok := freqToPIDs[params.Frequency]; ok {
				if len(params.PIDs) > len(existing) {
					freqToPIDs[params.Frequency] = params.PIDs
				}
			} else {
				freqToPIDs[params.Frequency] = params.PIDs
			}
		}
	}

	if len(freqToPIDs) == 0 {
		return nil, fmt.Errorf("no frequency entries found in channellist")
	}

	return freqToPIDs, nil
}

// FormatPIDsForDisplay converts a comma-separated PID string to a user-friendly
// hex representation, e.g. "0,400,401" -> "0x0000, 0x0190, 0x0191".
func FormatPIDsForDisplay(pids string) string {
	if pids == "" {
		return ""
	}

	parts := strings.Split(pids, ",")
	formatted := make([]string, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			formatted = append(formatted, p)
			continue
		}
		formatted = append(formatted, fmt.Sprintf("0x%04X", n))
	}
	return strings.Join(formatted, ", ")
}
