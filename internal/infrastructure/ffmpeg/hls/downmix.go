package hls

import (
	"fmt"
	"strings"
)

// DownMixAlgorithm represents a stereo downmix algorithm.
type DownMixAlgorithm string

const (
	DownMixNone    DownMixAlgorithm = "none"
	DownMixDave750 DownMixAlgorithm = "dave750"
	DownMixAC4     DownMixAlgorithm = "ac4"
)

// downmixFilters maps (algorithm, source channel count) to FFmpeg pan filter strings.
// Source: Jellyfin implementations from jellyfin/jellyfin#9001 and jellyfin/jellyfin#12354.
// Channel positions: c0=FL, c1=FR, c2=FC, c3=LFE, c4=BL, c5=BR, c6=SL, c7=SR.
// Only layouts requiring downmix (>2 channels) are included.
var downmixFilters = map[DownMixAlgorithm]map[int]string{
	DownMixDave750: {
		6: "pan=stereo|c0=0.5*c2+0.707*c0+0.707*c4+0.5*c3|c1=0.5*c2+0.707*c1+0.707*c5+0.5*c3",
		8: "pan=5.1(side)|c0=c0|c1=c1|c2=c2|c3=c3|c4=0.707*c4+0.707*c6|c5=0.707*c5+0.707*c7," +
			"pan=stereo|c0=0.5*c2+0.707*c0+0.707*c4+0.5*c3|c1=0.5*c2+0.707*c1+0.707*c5+0.5*c3",
	},
	DownMixAC4: {
		3: "pan=stereo|c0=c0+0.707*c2|c1=c1+0.707*c2",
		5: "pan=stereo|c0=c0+0.707*c2+0.707*c3|c1=c1+0.707*c2+0.707*c4",
		6: "pan=stereo|c0=c0+0.707*c2+0.707*c4|c1=c1+0.707*c2+0.707*c5",
		7: "pan=5.0(side)|c0=c0|c1=c1|c2=c2|c3=0.707*c3+0.707*c5|c4=0.707*c4+0.707*c6," +
			"pan=stereo|c0=c0+0.707*c2+0.707*c3|c1=c1+0.707*c2+0.707*c4",
		8: "pan=5.1(side)|c0=c0|c1=c1|c2=c2|c3=c3|c4=0.707*c4+0.707*c6|c5=0.707*c5+0.707*c7," +
			"pan=stereo|c0=c0+0.707*c2+0.707*c4|c1=c1+0.707*c2+0.707*c5",
	},
}

// buildDownmixFilterChain returns the complete -af filter string for downmixing
// multi-channel audio to stereo. Returns empty string if no downmix is needed.
//
// For non-None algorithms, the chain includes a pan filter derived from the selected
// algorithm's coefficients. If boost != 1.0, a volume filter is appended.
// For None algorithm, only the volume filter is added (FFmpeg uses its built-in
// ATSC A/52 downmix matrix via -ac 2).
func buildDownmixFilterChain(algorithm DownMixAlgorithm, sourceChannels int, boost float64) string {
	if sourceChannels <= 2 {
		return ""
	}

	var filters []string

	if algorithm != DownMixNone {
		if layouts, ok := downmixFilters[algorithm]; ok {
			if panFilter, ok := layouts[sourceChannels]; ok {
				filters = append(filters, panFilter)
			}
		}
	}

	if boost != 1.0 && boost > 0 {
		filters = append(filters, fmt.Sprintf("volume=%.4f", boost))
	}

	return strings.Join(filters, ",")
}
