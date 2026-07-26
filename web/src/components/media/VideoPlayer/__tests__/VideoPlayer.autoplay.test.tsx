/**
 * Tests for VideoPlayer auto-play / "Up Next" functionality.
 *
 * Behavior:
 * - When a video ends and autoplay is enabled (onAutoPlayNext provided),
 *   a 10-second countdown overlay appears.
 * - If the user takes no action, onAutoPlayNext is called when countdown hits 0.
 * - If the user clicks Cancel, the countdown stops and onAutoPlayCancel is called.
 * - When the next episode loads (mediaId changes), countdown state resets.
 * - autoPlayTriggeredRef prevents the ended event from re-firing the countdown
 *   during the episode transition window.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act, fireEvent, type RenderResult } from '@testing-library/react'
import { VideoPlayer } from '../VideoPlayer'
import type { VideoPlayerProps } from '../VideoPlayer.types'

// ── Mocks ───────────────────────────────────────────────────────────────────

const noopRef = { current: null }

vi.mock('@/lib/hooks/useHlsPlayer', () => ({
  useHlsPlayer: () => ({
    hlsRef: noopRef,
    streamOffsetRef: noopRef,
    availableQualities: [],
    currentQuality: null,
    currentBandwidth: null,
    availableAudioTracks: [],
    currentAudioTrack: -1,
    availableSubtitleTracks: [],
    currentSubtitleTrack: -1,
    changeQuality: () => null,
    changeAudioTrack: () => {},
    changeSubtitleTrack: () => {},
    loadSource: () => {},
  }),
}))

vi.mock('@/lib/hooks/useVideoControls', () => ({
  useVideoControls: () => ({
    handlePlayPause: vi.fn(),
    handleSeek: vi.fn(),
    handleVolumeChange: vi.fn(),
    handleMuteToggle: vi.fn(),
    handleFullscreenToggle: vi.fn(),
    handlePiPToggle: vi.fn(),
    handleSkip: vi.fn(),
    handlePlaybackSpeedChange: vi.fn(),
  }),
}))

vi.mock('@/lib/hooks/useVideoKeyboard', () => ({
  useVideoKeyboard: () => {},
}))

vi.mock('@/lib/hooks/useProgress', () => ({
  useProgressUpdater: () => ({
    startTracking: vi.fn(),
    stopTracking: vi.fn(),
    updateCurrentTime: vi.fn(),
    updatePreferences: vi.fn(),
  }),
}))

vi.mock('@/lib/hooks/useAutoQuality', () => ({
  useAutoQuality: () => ({
    recordSample: vi.fn(),
    recordStall: vi.fn(),
    networkStats: null,
  }),
}))

vi.mock('@/lib/hooks/usePlaybackAnalytics', () => ({
  usePlaybackAnalytics: () => ({
    startSession: vi.fn(),
    updateSessionId: vi.fn(),
    endSession: vi.fn(),
    recordQualitySwitch: vi.fn(),
    recordStall: vi.fn(),
    recordPlayTime: vi.fn(),
    recordStartupTime: vi.fn(),
  }),
}))

vi.mock('@/lib/hooks/useStreamStats', () => ({
  useStreamStats: () => ({
    stats: null,
    isLoading: false,
    refresh: vi.fn(),
  }),
}))

vi.mock('@/lib/hooks/useSubtitles', () => ({
  useSubtitles: () => ({
    availableSubtitles: [],
    currentSubtitle: null,
    setCurrentSubtitle: vi.fn(),
    textStreamIndex: -1,
    bitmapStreamIndex: -1,
  }),
}))

vi.mock('@/lib/api/generated/media/media', () => ({
  useGetApiMediaIdTracks: () => ({
    data: { status: 200, data: { subtitle_tracks: [], audio_tracks: [] } },
  }),
}))

vi.mock('@/lib/capabilities', () => ({
  getDeviceProfileHash: () => 'test-hash',
}))

vi.mock('@/utils/device', () => ({
  getAutoFullscreenPreference: () => false,
  enterFullscreen: vi.fn().mockResolvedValue(undefined),
  isInCSSFullscreen: () => false,
}))

// Capture onEnded and onPlay so tests can simulate video lifecycle events
let capturedOnEnded: (() => void) | null = null
let capturedOnPlay: (() => void) | null = null

vi.mock('@/lib/hooks/useVideoEvents', () => ({
  useVideoEvents: (opts: { onEnded: () => void; onPlay: () => void }) => {
    capturedOnEnded = opts.onEnded
    capturedOnPlay = opts.onPlay
  },
}))

vi.mock('../VideoControls', () => ({
  VideoControls: () => <div data-testid="video-controls" />,
}))

vi.mock('../StatsPanel', () => ({
  StatsPanel: () => null,
}))

vi.mock('../SubtitleOverlay', () => ({
  SubtitleOverlay: () => null,
}))

vi.mock('@/components/ui', () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}))

// ── Helpers ─────────────────────────────────────────────────────────────────

const defaultProps: VideoPlayerProps = {
  mediaId: 1,
  streamUrl: 'http://example.com/stream.m3u8',
  duration: 3600,
  metadata: { title: 'Test Episode', subtitle: 'S01:E01' },
  onClose: vi.fn(),
}

function renderPlayer(overrides: Partial<VideoPlayerProps> = {}): RenderResult {
  return render(<VideoPlayer {...defaultProps} {...overrides} />)
}

/** Advance fake timers one second at a time, flushing React updates each tick. */
function advanceSeconds(n: number) {
  for (let i = 0; i < n; i++) {
    act(() => { vi.advanceTimersByTime(1000) })
  }
}

/** Check that the countdown number matches. The number and "s" are separate text nodes. */
function getCountdownValue(): number | null {
  const el = screen.queryByText((content, node) => {
    return node?.tagName === 'P' && /^\d+s$/.test(node.textContent || '')
  })
  if (!el) return null
  const text = el.textContent || ''
  return parseInt(text.replace('s', ''), 10)
}

// ── Tests ───────────────────────────────────────────────────────────────────

describe('VideoPlayer auto-play', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    capturedOnEnded = null
    capturedOnPlay = null
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders without crashing', () => {
    renderPlayer()
    expect(screen.getByTestId('video-controls')).toBeInTheDocument()
  })

  it('does not show the up-next overlay when onAutoPlayNext is not provided', () => {
    renderPlayer()
    act(() => { capturedOnEnded?.() })
    advanceSeconds(1)
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()
  })

  it('starts a 10-second countdown when the video ends and onAutoPlayNext is provided', () => {
    renderPlayer({
      onAutoPlayNext: vi.fn(),
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    act(() => { capturedOnEnded?.() })

    expect(screen.getByText('Next episode')).toBeInTheDocument()
    expect(getCountdownValue()).toBe(10)
  })

  it('counts down from 10 to 0 and calls onAutoPlayNext', () => {
    const onAutoPlayNext = vi.fn()
    renderPlayer({
      onAutoPlayNext,
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    // Video ends → countdown starts at 10
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)

    // After 5 seconds, countdown is at 5
    advanceSeconds(5)
    expect(getCountdownValue()).toBe(5)

    // After 10 seconds total, countdown reaches 0 and onAutoPlayNext fires
    advanceSeconds(5)
    expect(onAutoPlayNext).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()
  })

  it('cancel button stops the countdown and calls onAutoPlayCancel', () => {
    const onAutoPlayNext = vi.fn()
    const onAutoPlayCancel = vi.fn()
    renderPlayer({
      onAutoPlayNext,
      onAutoPlayCancel,
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    // Video ends → countdown starts
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)

    // Click Cancel (use fireEvent to avoid async issues with fake timers)
    act(() => { fireEvent.click(screen.getByText('Cancel')) })

    // onAutoPlayNext should NOT have been called
    expect(onAutoPlayNext).not.toHaveBeenCalled()
    // onAutoPlayCancel SHOULD have been called
    expect(onAutoPlayCancel).toHaveBeenCalledTimes(1)
    // Overlay should be gone
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()

    // Even after full 10 seconds, auto-play does not fire
    advanceSeconds(10)
    expect(onAutoPlayNext).not.toHaveBeenCalled()
  })

  it('resets countdown when mediaId changes (new episode loads)', () => {
    const onAutoPlayNext = vi.fn()
    const { rerender } = renderPlayer({
      mediaId: 1,
      onAutoPlayNext,
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    // Episode 1 ends → countdown starts
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)

    // Advance partway
    advanceSeconds(3)
    expect(getCountdownValue()).toBe(7)

    // Simulate episode change
    capturedOnEnded = null
    rerender(
      <VideoPlayer
        {...defaultProps}
        mediaId={2}
        nextEpisodeInfo={{ title: 'Ep 3', season: 1, episode: 3 }}
        onAutoPlayNext={onAutoPlayNext}
      />
    )

    // Countdown overlay should be gone (autoPlayCountdown reset to null)
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()

    // Advance more time — countdown should NOT continue
    advanceSeconds(10)
    expect(onAutoPlayNext).not.toHaveBeenCalled()
  })

  it('allows countdown to start again for the new episode after reset', () => {
    const onAutoPlayNext = vi.fn()
    const { rerender } = renderPlayer({
      mediaId: 1,
      onAutoPlayNext,
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    // Episode 1 ends → countdown → auto-play fires
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)
    advanceSeconds(10)
    expect(onAutoPlayNext).toHaveBeenCalledTimes(1)

    // Simulate new episode loading
    capturedOnEnded = null
    capturedOnPlay = null
    rerender(
      <VideoPlayer
        {...defaultProps}
        mediaId={2}
        onAutoPlayNext={onAutoPlayNext}
        nextEpisodeInfo={{ title: 'Ep 3', season: 1, episode: 3 }}
      />
    )

    // No overlay during normal playback
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()

    // Simulate the new video starting to play (resets autoPlayTriggeredRef)
    act(() => { capturedOnPlay?.() })

    // Episode 2 ends → new countdown
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)

    // New countdown completes → auto-play fires again
    advanceSeconds(10)
    expect(onAutoPlayNext).toHaveBeenCalledTimes(2)
  })

  it('does not re-trigger countdown during episode transition (autoPlayTriggeredRef guard)', () => {
    const onAutoPlayNext = vi.fn()
    renderPlayer({
      mediaId: 1,
      onAutoPlayNext,
      nextEpisodeInfo: { title: 'Ep 2', season: 1, episode: 2 },
    })

    // Video ends → countdown starts
    act(() => { capturedOnEnded?.() })
    expect(getCountdownValue()).toBe(10)

    // Countdown reaches 0 → auto-play fires, overlay hidden
    advanceSeconds(10)
    expect(onAutoPlayNext).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()

    // Simulate spurious ended event during transition
    // (autoPlayTriggeredRef is true, so this should be blocked)
    act(() => { capturedOnEnded?.() })

    // The overlay should NOT reappear
    expect(screen.queryByText('Next episode')).not.toBeInTheDocument()
  })
})
