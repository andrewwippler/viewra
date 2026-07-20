package hls

import (
	"strings"
	"testing"
)

func TestBuildDownmixFilterChain(t *testing.T) {
	tests := []struct {
		name          string
		algorithm     DownMixAlgorithm
		sourceChans   int
		boost         float64
		wantEmpty     bool
		wantContains  []string // substrings that must appear in the result
		wantNotContain []string // substrings that must NOT appear
	}{
		{
			name:      "stereo source - no filter needed",
			algorithm: DownMixAC4,
			sourceChans: 2,
			boost:     2.0,
			wantEmpty: true,
		},
		{
			name:      "mono source - no filter needed",
			algorithm: DownMixAC4,
			sourceChans: 1,
			boost:     2.0,
			wantEmpty: true,
		},
		{
			name:        "AC4 5.1 - pan filter present",
			algorithm:   DownMixAC4,
			sourceChans: 6,
			boost:       2.0,
			wantContains: []string{
				"pan=stereo|c0=c0+0.707*c2+0.707*c4|c1=c1+0.707*c2+0.707*c5",
				"volume=",
			},
		},
		{
			name:        "AC4 7.1 - cascaded pan filter",
			algorithm:   DownMixAC4,
			sourceChans: 8,
			boost:       2.0,
			wantContains: []string{
				"pan=5.1(side)",
				"pan=stereo|c0=c0+0.707*c2+0.707*c4|c1=c1+0.707*c2+0.707*c5",
				"volume=",
			},
		},
		{
			name:        "AC4 3.0 - simple pan",
			algorithm:   DownMixAC4,
			sourceChans: 3,
			boost:       2.0,
			wantContains: []string{
				"pan=stereo|c0=c0+0.707*c2|c1=c1+0.707*c2",
				"volume=",
			},
		},
		{
			name:        "AC4 5.0 - pan with surround",
			algorithm:   DownMixAC4,
			sourceChans: 5,
			boost:       2.0,
			wantContains: []string{
				"pan=stereo|c0=c0+0.707*c2+0.707*c3|c1=c1+0.707*c2+0.707*c4",
				"volume=",
			},
		},
		{
			name:        "AC4 7.0 - cascaded pan",
			algorithm:   DownMixAC4,
			sourceChans: 7,
			boost:       2.0,
			wantContains: []string{
				"pan=5.0(side)",
				"volume=",
			},
		},
		{
			name:        "Dave750 5.1 - dave750 pan",
			algorithm:   DownMixDave750,
			sourceChans: 6,
			boost:       4.25,
			wantContains: []string{
				"pan=stereo|c0=0.5*c2+0.707*c0+0.707*c4+0.5*c3|c1=0.5*c2+0.707*c1+0.707*c5+0.5*c3",
				"volume=4.2500",
			},
		},
		{
			name:        "Dave750 7.1 - cascaded dave750",
			algorithm:   DownMixDave750,
			sourceChans: 8,
			boost:       4.25,
			wantContains: []string{
				"pan=5.1(side)",
				"pan=stereo|c0=0.5*c2",
				"volume=4.2500",
			},
		},
		{
			name:        "None 5.1 - volume only, no pan",
			algorithm:   DownMixNone,
			sourceChans: 6,
			boost:       2.0,
			wantContains: []string{
				"volume=2.0000",
			},
			wantNotContain: []string{"pan="},
		},
		{
			name:        "None 5.1 boost 1.0 - empty filter",
			algorithm:   DownMixNone,
			sourceChans: 6,
			boost:       1.0,
			wantEmpty:   true,
		},
		{
			name:        "AC4 5.1 boost 1.0 - pan only, no volume",
			algorithm:   DownMixAC4,
			sourceChans: 6,
			boost:       1.0,
			wantContains: []string{
				"pan=stereo|c0=c0+0.707*c2+0.707*c4|c1=c1+0.707*c2+0.707*c5",
			},
			wantNotContain: []string{"volume="},
		},
		{
			name:        "unknown algorithm with boost - volume only",
			algorithm:   DownMixAlgorithm("unknown"),
			sourceChans: 6,
			boost:       1.5,
			wantContains: []string{
				"volume=1.5000",
			},
			wantNotContain: []string{"pan="},
		},
		{
			name:        "unknown algorithm boost 1.0 - empty",
			algorithm:   DownMixAlgorithm("unknown"),
			sourceChans: 6,
			boost:       1.0,
			wantEmpty:   true,
		},
		{
			name:        "4 channel source with AC4 - falls back to volume only",
			algorithm:   DownMixAC4,
			sourceChans: 4,
			boost:       2.0,
			wantContains: []string{
				"volume=2.0000",
			},
			wantNotContain: []string{"pan="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildDownmixFilterChain(tt.algorithm, tt.sourceChans, tt.boost)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("buildDownmixFilterChain() = %q, want empty", got)
				}
				return
			}

			if got == "" {
				t.Fatalf("buildDownmixFilterChain() = empty, want non-empty filter chain")
			}

			for _, substr := range tt.wantContains {
				if !strings.Contains(got, substr) {
					t.Errorf("buildDownmixFilterChain() = %q, missing %q", got, substr)
				}
			}

			for _, substr := range tt.wantNotContain {
				if strings.Contains(got, substr) {
					t.Errorf("buildDownmixFilterChain() = %q, should not contain %q", got, substr)
				}
			}
		})
	}
}

func TestBuildDownmixFilterChainPanCoefficients(t *testing.T) {
	// Verify the pan filter strings are valid FFmpeg filter syntax
	// by checking they contain the expected channel mapping patterns
	algorithms := []DownMixAlgorithm{DownMixDave750, DownMixAC4}

	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			layouts, ok := downmixFilters[algo]
			if !ok {
				t.Fatalf("algorithm %s not found in downmixFilters", algo)
			}

			for chans, filter := range layouts {
				if chans < 3 || chans > 8 {
					t.Errorf("unexpected channel count %d for algorithm %s", chans, algo)
				}
				if !strings.HasPrefix(filter, "pan=") {
					t.Errorf("filter for %s/%dch does not start with 'pan=': %s", algo, chans, filter)
				}
				if !strings.Contains(filter, "c0=") || !strings.Contains(filter, "c1=") {
					t.Errorf("filter for %s/%dch missing c0 or c1 output: %s", algo, chans, filter)
				}
			}
		})
	}
}
