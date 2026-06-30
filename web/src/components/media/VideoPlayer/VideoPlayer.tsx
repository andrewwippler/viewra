/**
 * VideoPlayer Component
 * Full-featured video player with HLS streaming, quality selection,
 * keyboard controls, and progress tracking.
 *
 * Quality Control Model (Single-Quality):
 * - Backend picks optimal quality based on client capabilities
 * - User can override via quality picker (triggers stream reload)
 * - Each quality change restarts FFmpeg from current position
 */

import { Button } from '@/components/ui'
import { useProgressUpdater } from '@/lib/hooks/useProgress'
import { useAutoQuality } from '@/lib/hooks/useAutoQuality'
import { usePlaybackAnalytics } from '@/lib/hooks/usePlaybackAnalytics'
import { useStreamStats } from '@/lib/hooks/useStreamStats'
import { useHlsPlayer } from '@/lib/hooks/useHlsPlayer'
import { useVideoEvents } from '@/lib/hooks/useVideoEvents'
import { useVideoKeyboard } from '@/lib/hooks/useVideoKeyboard'
import { useVideoControls } from '@/lib/hooks/useVideoControls'
import { useSubtitles } from '@/lib/hooks/useSubtitles'
import { useCallback, useEffect, useRef, useState } from 'react'
import { VideoControls } from './VideoControls'
import { StatsPanel } from './StatsPanel'
import { SubtitleOverlay } from './SubtitleOverlay'
import type { VideoPlayerProps } from './VideoPlayer.types'
import { useGetApiMediaIdTracks } from '@/lib/api/generated/media/media'
import { getDeviceProfileHash } from '@/lib/capabilities'
import { getAutoFullscreenPreference, enterFullscreen, isInCSSFullscreen } from '@/utils/device'

export const VideoPlayer = ({
  mediaId,
  streamUrl,
  initialPosition = 0,
  duration = 0,
  metadata,
  onClose,
  onTimeUpdate,
  availableQualities: backendQualities = [],
  selectedQualityId = null,
  onQualityChange: onQualityChangeCallback,
  savedPreferences = null,
  nextEpisodeInfo,
  onAutoPlayNext,
  onAutoPlayCancel,
  onPlayNext,
  onPlayPrev,
}: VideoPlayerProps) => {
  const videoRef = useRef<HTMLVideoElement>(null)
  const videoContainerRef = useRef<HTMLDivElement>(null)
  const isSeekingRef = useRef<boolean>(false)

  // Detect if this is an HLS stream
  const isHlsStream = streamUrl.includes('.m3u8')

  // Core playback state
  const [isPlaying, setIsPlaying] = useState(false)
  const [videoDuration, setVideoDuration] = useState(duration)
  const [error, setError] = useState<string | null>(null)
  const [isBuffering, setIsBuffering] = useState<boolean>(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [volume, setVolume] = useState(1)
  const [isMuted, setIsMuted] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [isPiP, setIsPiP] = useState(false)
  const [playbackSpeed, setPlaybackSpeed] = useState<number>(1)
  const [showDebugOverlay, setShowDebugOverlay] = useState(false)
  // Key that changes on large seeks to force subtitle components to remount
  // This ensures subtitles re-initialize with the correct streamOffsetRef value
  const [subtitleSeekKey, setSubtitleSeekKey] = useState(0)
  // Backend session ID for analytics correlation (from X-Session-ID header)
  const [backendSessionId, setBackendSessionId] = useState<string | null>(null)

  // Auto-play countdown state
  const [autoPlayCountdown, setAutoPlayCountdown] = useState<number | null>(null)
  const autoPlayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoPlayCountdownShownRef = useRef(false)

  const clearAutoPlayTimer = useCallback(() => {
    if (autoPlayTimerRef.current) {
      clearTimeout(autoPlayTimerRef.current)
      autoPlayTimerRef.current = null
    }
  }, [])

  // Fetch tracks (audio and subtitle) from API
  const { data: tracksData } = useGetApiMediaIdTracks(mediaId)
  const subtitleTracksFromApi =
    tracksData?.status === 200 ? tracksData.data.subtitle_tracks || [] : []
  const audioTracksFromApi =
    tracksData?.status === 200 ? tracksData.data.audio_tracks || [] : []

  // Track current audio stream index for URL construction
  // -1 means default (first audio track)
  const [currentAudioStreamIndex, setCurrentAudioStreamIndex] = useState<number>(-1)

  // Position override for when we switch audio tracks or quality - maintains playback position
  const [trackSwitchPosition, setTrackSwitchPosition] = useState<number | null>(null)

  // Build stream URL with audio track parameter and potentially updated start position
  // When quality changes to a different resolution, we need to reload with the current position
  const buildEffectiveStreamUrl = () => {
    let url = streamUrl
    const params = new URLSearchParams()

    // Parse existing params from URL
    const urlParts = streamUrl.split('?')
    if (urlParts.length > 1) {
      const existingParams = new URLSearchParams(urlParts[1])
      existingParams.forEach((value, key) => params.set(key, value))
      url = urlParts[0]
    }

    // Update audio track if non-default
    if (currentAudioStreamIndex > 0) {
      params.set('audioTrack', String(currentAudioStreamIndex))
    }

    // Update start position if we're doing a track/quality switch
    if (trackSwitchPosition !== null) {
      params.set('start', String(Math.floor(trackSwitchPosition)))
    }

    const paramStr = params.toString()
    return paramStr ? `${url}?${paramStr}` : url
  }

  const effectiveStreamUrl = buildEffectiveStreamUrl()

  // Clear the track switch position once the stream has successfully shifted and resumed
  useEffect(() => {
    if (trackSwitchPosition !== null && isPlaying) {
      setTrackSwitchPosition(null)
    }
  }, [effectiveStreamUrl, isPlaying, trackSwitchPosition])

  // Use track switch position if set (from audio or quality change), otherwise use initial position from props
  const effectiveInitialPosition = trackSwitchPosition ?? initialPosition

  // Subtitle track management
  // Skip auto-selection when we have a saved subtitle preference to restore
  // Default subtitles to off (user can enable manually or via saved preference)
  const hasSavedSubtitlePref = savedPreferences?.selectedSubtitleTrack !== undefined
  const { availableSubtitles, currentSubtitle, setCurrentSubtitle, textStreamIndex, bitmapStreamIndex } = useSubtitles({
    subtitleTracks: subtitleTracksFromApi,
    preferredLanguage: 'off',
    preferSDH: false,
    preferForced: true,
    skipAutoSelect: hasSavedSubtitlePref,
  })

  // Initialize progress updater
  const progressUpdater = useProgressUpdater(mediaId, videoDuration)

  // HLS player hook - handles HLS.js lifecycle, quality
  // Uses Navigator Connection API for instant bandwidth estimate (no slow speed test)
  // Audio tracks are managed via API, not HLS.js (since audio is muxed into video segments)
  const {
    hlsRef,
    availableQualities: _availableQualities,
    currentQuality: _currentQuality,
    currentBandwidth: _currentBandwidth,
    streamOffsetRef,
    changeQuality: _changeQuality,
  } = useHlsPlayer({
    videoRef,
    streamUrl: effectiveStreamUrl,
    initialPosition: effectiveInitialPosition,
    isHlsStream,
    onError: setError,
    onFragLoaded: (bytes, durationMs) => recordSample(bytes, durationMs),
    onSessionIdReceived: setBackendSessionId,
  })

  // Convert API audio tracks to the format expected by VideoControls
  // The id is the stream_index which is what we pass to the backend
  // Filter out tracks without stream_index (shouldn't happen, but type-safe)
  const availableAudioTracks = audioTracksFromApi
    .filter((track): track is typeof track & { stream_index: number } =>
      track.stream_index !== undefined
    )
    .map((track, index) => ({
      id: track.stream_index,
      name: track.title || `Track ${index + 1}`,
      language: track.language || 'Unknown',
    }))

  // Current audio track for UI - this is the stream_index (matches track.id)
  // Default to first track's stream_index if not explicitly set
  const currentAudioTrack = currentAudioStreamIndex > 0
    ? currentAudioStreamIndex
    : (availableAudioTracks[0]?.id ?? 0)

  // Network stats for debug panel
  const { recordSample, recordStall, networkStats } = useAutoQuality({
    enabled: isHlsStream,
    hlsInstance: hlsRef.current,
  })

  // Stream stats for debug panel
  const { stats: streamStats, isLoading: streamStatsLoading, refresh: refreshStreamStats } = useStreamStats({
    mediaId,
    videoRef,
    hlsRef,
    networkStats,
    isPlaying,
    playbackMode: isHlsStream ? 'transcode' : 'direct',
    selectedQualityId,  // Pass selected quality for accurate strategy detection
    streamOffset: streamOffsetRef.current || 0,
    enabled: showDebugOverlay,
  })

  // Playback analytics
  const {
    startSession,
    updateSessionId,
    endSession,
    recordQualitySwitch,
    recordStall: recordAnalyticsStall,
    recordPlayTime,
    recordStartupTime,
  } = usePlaybackAnalytics({ enabled: true })

  // Update analytics session ID when backend session ID arrives
  useEffect(() => {
    if (backendSessionId) {
      updateSessionId(backendSessionId)
    }
  }, [backendSessionId, updateSessionId])

  // Video controls hook
  const {
    handlePlayPause,
    handleSeek,
    handleVolumeChange: handleVolumeChangeControl,
    handleMuteToggle,
    handleFullscreenToggle,
    handlePiPToggle,
    handleSkip,
    handlePlaybackSpeedChange,
  } = useVideoControls({
    videoRef,
    containerRef: videoContainerRef,
    hlsRef,
    streamOffsetRef,
    isSeekingRef,
    isHlsStream,
    videoDuration,
    progressUpdater,
    onTimeUpdate: (time) => {
      setCurrentTime(time)
      if (onTimeUpdate) {
        onTimeUpdate(time)
      }
    },
    onLargeSeekComplete: () => {
      // Force subtitle components to remount with fresh streamOffsetRef value
      setSubtitleSeekKey((prev) => prev + 1)
    },
  })

  // Video events hook
  useVideoEvents({
    videoRef,
    mediaId,
    duration,
    videoDuration,
    isPlaying,
    streamOffsetRef,
    isSeekingRef,
    backendSessionId,
    onPlay: () => setIsPlaying(true),
    onPause: () => setIsPlaying(false),
    onTimeUpdate: (time) => {
      setCurrentTime(time)
      if (onTimeUpdate) {
        onTimeUpdate(time)
      }
    },
    onEnded: () => {
      setIsPlaying(false)
      endSession()
      // Fallback: start countdown if not already triggered by near-end detection
      if (onAutoPlayNext && !autoPlayCountdownShownRef.current) {
        autoPlayCountdownShownRef.current = true
        setAutoPlayCountdown(10)
      }
    },
    onBufferingStart: () => setIsBuffering(true),
    onBufferingEnd: (stallDuration) => {
      setIsBuffering(false)
      if (stallDuration > 100) {
        recordAnalyticsStall(stallDuration)
      }
    },
    onVolumeChange: (vol, muted) => {
      setVolume(vol)
      setIsMuted(muted)
    },
    onFullscreenChange: setIsFullscreen,
    onPiPEnter: () => setIsPiP(true),
    onPiPExit: () => setIsPiP(false),
    onDurationChange: setVideoDuration,
    startAnalyticsSession: startSession,
    recordStall,
    recordPlayTime,
    recordStartupTime,
    progressUpdater,
  })

  // Keyboard shortcuts hook
  useVideoKeyboard({
    videoRef,
    containerRef: videoContainerRef,
    videoDuration,
    onToggleDebug: () => setShowDebugOverlay((prev) => !prev),
  })

  // TV mode: handle WebOS remote media keys and Back/Exit button
  // Uses capture phase on document to catch events before they reach the
  // bubble phase, since WebOS Chrome 79 may not reliably bubble media keys
  useEffect(() => {
    if (!__TV_MODE__) {return}

    const handleKeyDown = (e: KeyboardEvent) => {
      const video = videoRef.current
      if (!video) {return}
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {return}

      switch (e.keyCode) {
        case 461: // Back/Exit on WebOS remote
          if (onClose) {
            e.preventDefault()
            e.stopPropagation()
            onClose()
          }
          return
        case 179:
        case 413:
        case 415: // Play/Pause
          e.preventDefault()
          e.stopPropagation()
          if (video.paused) { video.play() } else { video.pause() }
          return
        case 412:
        case 464: // Rewind 10 seconds
          e.preventDefault()
          e.stopPropagation()
          video.currentTime = Math.max(0, video.currentTime - 10)
          return
        case 417:
        case 465: // Fast Forward 10 seconds
          e.preventDefault()
          e.stopPropagation()
          video.currentTime = Math.min(video.duration || videoDuration, video.currentTime + 10)
          return
        case 13: // Enter/OK on WebOS remote — on countdown overlay, this activates Play Now
          if (autoPlayCountdown !== null) {
            e.preventDefault()
            e.stopPropagation()
            clearAutoPlayTimer()
            autoPlayNextRef.current?.()
            setAutoPlayCountdown(null)
          }
          return
      }
    }

    document.addEventListener('keydown', handleKeyDown, { capture: true })
    return () => document.removeEventListener('keydown', handleKeyDown, { capture: true })
  }, [onClose, videoDuration, autoPlayCountdown, clearAutoPlayTimer])

  // WebOS Media Session API: routes remote media keys to the video player
  // when the platform intercepts them before generating keyboard events
  useEffect(() => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {return}

    const video = videoRef.current
    if (!video) {return}

    navigator.mediaSession.setActionHandler('play', () => { video.play() })
    navigator.mediaSession.setActionHandler('pause', () => { video.pause() })
    navigator.mediaSession.setActionHandler('seekbackward', (details) => {
      const skip = details.seekOffset || 10
      video.currentTime = Math.max(0, video.currentTime - skip)
    })
    navigator.mediaSession.setActionHandler('seekforward', (details) => {
      const skip = details.seekOffset || 10
      video.currentTime = Math.min(video.duration || 0, video.currentTime + skip)
    })

    return () => {
      navigator.mediaSession.setActionHandler('play', null)
      navigator.mediaSession.setActionHandler('pause', null)
      navigator.mediaSession.setActionHandler('seekbackward', null)
      navigator.mediaSession.setActionHandler('seekforward', null)
    }
  }, [])

  // Browser close progress save (includes preferences and device profile)
  // Note: -1 is used as a sentinel value for "subtitles off" to distinguish from null (don't update)
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (currentTime > 0 && videoDuration > 0) {
        const data = JSON.stringify({
          media_id: mediaId,
          user_id: 1,
          progress_seconds: currentTime,
          duration_seconds: videoDuration,
          device_profile: getDeviceProfileHash(),
          selected_quality: selectedQualityId,
          selected_audio_track: currentAudioStreamIndex > 0 ? currentAudioStreamIndex : null,
          // Use -1 to indicate "subtitles off" (null means don't update due to COALESCE in SQL)
          selected_subtitle_track: currentSubtitle?.id ?? -1,
        })
        const apiUrl = `${window.location.origin}/api/progress`
        navigator.sendBeacon(apiUrl, new Blob([data], { type: 'application/json' }))
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [mediaId, currentTime, videoDuration, selectedQualityId, currentAudioStreamIndex, currentSubtitle])

  // Auto-fullscreen when video starts playing
  useEffect(() => {
    const video = videoRef.current
    const container = videoContainerRef.current
    if (!video || !container) {return}

    const autoFullscreenEnabled = getAutoFullscreenPreference()
    // 1. User preference enables it
    // 2. We're on WebOS TV or user explicitly enabled it
    // 3. Video is playing
    if (!autoFullscreenEnabled || !isPlaying) {return}

    let cancelled = false

    // Small delay to ensure video metadata is loaded
    const timer = setTimeout(() => {
      if (cancelled) {return}

      // Check if already in fullscreen (native or CSS fallback)
      if (document.fullscreenElement || isInCSSFullscreen(container)) {
        return
      }

      // Try to enter fullscreen
      const fullscreenPromise = enterFullscreen(container)

      fullscreenPromise.catch(() => {
        // If native fullscreen fails, CSS fallback is applied by enterFullscreen
        // The CSS fallback adds .fullscreen-fallback class
      })
    }, 500)

    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [isPlaying])

  // Pause heartbeat: Send keepalive every 20s while paused to prevent
  // the transcode session from being cleaned up by the idle timeout.
  // Session idle timeout is 5m (configurable), but we heartbeat at 20s to stay well within.
  useEffect(() => {
    const heartbeat = async () => {
      if (!selectedQualityId) {return}
      const params = new URLSearchParams({ quality: selectedQualityId })
      if (currentAudioStreamIndex > 0) {
        params.set('audioTrack', String(currentAudioStreamIndex))
      }
      try {
        await fetch(`/api/media/${mediaId}/hls/heartbeat?${params}`, {
          credentials: 'include',
        })
      } catch {
        // Silently ignore heartbeat failures
      }
    }

    const interval = setInterval(() => {
      if (!isPlaying) {
        heartbeat()
      }
    }, 20 * 1000)

    return () => clearInterval(interval)
  }, [mediaId, selectedQualityId, currentAudioStreamIndex, isPlaying])

  // Auto-play countdown effect: decrement each second, fire onAutoPlayNext when 0
  const autoPlayNextRef = useRef(onAutoPlayNext)
  autoPlayNextRef.current = onAutoPlayNext
  const autoPlayCancelRef = useRef(onAutoPlayCancel)
  autoPlayCancelRef.current = onAutoPlayCancel

  // Reset time-based state when switching episodes to prevent stale currentTime
  // from triggering auto-play with the new episode's duration
  useEffect(() => {
    clearAutoPlayTimer()
    setCurrentTime(0)
    setAutoPlayCountdown(null)
    autoPlayCountdownShownRef.current = false
  }, [mediaId, clearAutoPlayTimer])

  // Time-based trigger: show countdown overlay near the end of the episode
  // The useEffect on autoPlayCountdown handles all timing (decrement + firing)
  const UP_NEXT_THRESHOLD = 20
  useEffect(() => {
    if (!onAutoPlayNext) {return}
    if (autoPlayCountdownShownRef.current) {return}
    if (videoDuration <= 0 || currentTime <= 0) {return}
    const remaining = videoDuration - currentTime
    if (remaining > UP_NEXT_THRESHOLD || remaining <= 0) {return}

    autoPlayCountdownShownRef.current = true
    setAutoPlayCountdown(10)
  }, [currentTime, videoDuration, onAutoPlayNext])

  // Single countdown timer: decrement each second, fire onAutoPlayNext at 0
  useEffect(() => {
    if (autoPlayCountdown === null) {
      clearAutoPlayTimer()
      return
    }
    if (autoPlayCountdown <= 0) {
      clearAutoPlayTimer()
      autoPlayNextRef.current?.()
      setAutoPlayCountdown(null)
      return
    }
    const timer = setTimeout(() => {
      setAutoPlayCountdown((prev) => (prev ?? 0) - 1)
    }, 1000)
    autoPlayTimerRef.current = timer
    return () => {
      clearTimeout(timer)
    }
  }, [autoPlayCountdown, clearAutoPlayTimer])

  // TV mode: focus the play-now button when countdown appears
  useEffect(() => {
    if (__TV_MODE__ && autoPlayCountdown !== null) {
      const btn = document.getElementById('play-now-btn')
      btn?.focus()
    }
  }, [autoPlayCountdown])

  const handleCancelAutoPlay = () => {
    clearAutoPlayTimer()
    setAutoPlayCountdown(null)
    autoPlayCancelRef.current?.()
  }

  const handlePlayNow = () => {
    clearAutoPlayTimer()
    autoPlayNextRef.current?.()
    setAutoPlayCountdown(null)
  }

  const handlePlayNext = () => {
    clearAutoPlayTimer()
    setAutoPlayCountdown(null)
    autoPlayCountdownShownRef.current = false
    onPlayNext?.()
  }

  // Apply saved audio track preference when tracks load
  // Use a ref to ensure we only apply once per media
  const appliedAudioPrefRef = useRef(false)
  useEffect(() => {
    if (appliedAudioPrefRef.current) {return}
    if (!savedPreferences?.selectedAudioTrack) {return}
    if (availableAudioTracks.length === 0) {return}

    // Check if saved track exists in available tracks
    const trackExists = availableAudioTracks.some(t => t.id === savedPreferences.selectedAudioTrack)
    if (trackExists && savedPreferences.selectedAudioTrack !== currentAudioStreamIndex) {
      setCurrentAudioStreamIndex(savedPreferences.selectedAudioTrack)
    }
    appliedAudioPrefRef.current = true
  }, [savedPreferences, availableAudioTracks, currentAudioStreamIndex])

  // Apply saved subtitle preference when subtitles load
  const appliedSubtitlePrefRef = useRef(false)
  useEffect(() => {
    if (appliedSubtitlePrefRef.current) {return}
    if (savedPreferences?.selectedSubtitleTrack === undefined) {return}
    if (availableSubtitles.length === 0 && savedPreferences.selectedSubtitleTrack !== null) {return}

    // -1 or null means subtitles off, otherwise find the track
    if (savedPreferences.selectedSubtitleTrack === null || savedPreferences.selectedSubtitleTrack === -1) {
      if (currentSubtitle !== null) {
        setCurrentSubtitle(null)
      }
    } else {
      const trackExists = availableSubtitles.some(s => s.id === savedPreferences.selectedSubtitleTrack)
      if (trackExists && currentSubtitle?.id !== savedPreferences.selectedSubtitleTrack) {
        setCurrentSubtitle(savedPreferences.selectedSubtitleTrack)
      }
    }
    appliedSubtitlePrefRef.current = true
  }, [savedPreferences, availableSubtitles, currentSubtitle, setCurrentSubtitle])

  // Reset applied prefs refs when media changes
  useEffect(() => {
    appliedAudioPrefRef.current = false
    appliedSubtitlePrefRef.current = false
  }, [mediaId])

  // Save preferences when they change (via progress updater)
  // Use -1 to indicate "subtitles off" (null means don't update due to COALESCE in SQL)
  useEffect(() => {
    progressUpdater.updatePreferences({
      selectedQuality: selectedQualityId,
      selectedAudioTrack: currentAudioStreamIndex > 0 ? currentAudioStreamIndex : null,
      selectedSubtitleTrack: currentSubtitle?.id ?? -1,
    })
  }, [selectedQualityId, currentAudioStreamIndex, currentSubtitle, progressUpdater])

  // Handle quality change - calls parent callback to rebuild URL and reload stream
  // Single-quality model: each quality change triggers a new FFmpeg session from current position
  const handleQualityChange = useCallback(
    (qualityId: string) => {
      const video = videoRef.current
      if (!video || !onQualityChangeCallback) {return}

      // Calculate the actual media time (accounting for stream offset in progressive transcoding)
      const currentPosition = video.currentTime + (streamOffsetRef.current || 0)

      // Record quality switch for analytics
      recordQualitySwitch(
        selectedQualityId,
        qualityId,
        'user_manual',
        currentPosition,
        null,
        null
      )

      // Call parent callback to rebuild URL with ?quality= and reload
      onQualityChangeCallback(qualityId, currentPosition)

      // Refresh stream stats after a short delay to get updated strategy info
      setTimeout(() => refreshStreamStats(), 1000)
    },
    [onQualityChangeCallback, selectedQualityId, recordQualitySwitch, streamOffsetRef, refreshStreamStats]
  )

  // Handle audio track change
  // streamIndex is the FFmpeg stream index (passed as track.id from AudioSelector)
  const handleAudioTrackChange = useCallback(
    (streamIndex: number) => {
      // Capture current playback position before switching
      const video = videoRef.current
      if (video) {
        // Calculate the actual media time (accounting for stream offset in progressive transcoding)
        const actualTime = video.currentTime + (streamOffsetRef.current || 0)
        setTrackSwitchPosition(actualTime)
      }
      // Setting stream index will update effectiveStreamUrl via state change
      // which triggers HLS.js to reload with the new audio track
      setCurrentAudioStreamIndex(streamIndex)
    },
    [streamOffsetRef]
  )

  // Handle playback speed with state update
  const handleSpeedChange = useCallback(
    (speed: number) => {
      handlePlaybackSpeedChange(speed)
      setPlaybackSpeed(speed)
    },
    [handlePlaybackSpeedChange]
  )

  // Handle subtitle selection
  const handleSubtitleChange = useCallback(
    (trackId: number | null) => {
      setCurrentSubtitle(trackId)
    },
    [setCurrentSubtitle]
  )

  return (
    <div className="fixed inset-0 z-50 bg-black flex flex-col">
      {/* Close button */}
      <div className="absolute top-4 right-4 z-30">
        {onClose && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="text-white hover:bg-white/20 cursor-pointer"
          >
            Close
          </Button>
        )}
      </div>

      {/* Error display */}
      {error && (
        <div className="absolute top-20 left-1/2 transform -translate-x-1/2 z-20 bg-red-600/90 text-white px-6 py-3 rounded-lg shadow-lg max-w-md">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

      {/* Buffering indicator */}
      {isBuffering && (
        <div className="absolute inset-0 z-20 flex items-center justify-center pointer-events-none">
          <div
            className="w-16 h-16 rounded-full animate-spin"
            style={{
              border: '4px solid transparent',
              borderTopColor: 'white',
              borderRightColor: 'rgba(255, 255, 255, 0.3)',
              borderBottomColor: 'rgba(255, 255, 255, 0.1)',
            }}
          />
        </div>
      )}

      {/* Video player */}
      <div
        ref={videoContainerRef}
        className="flex-1 flex items-center justify-center bg-black relative"
      >
        <video
          ref={videoRef}
          className="w-full h-full max-h-screen cursor-pointer"
          style={{ objectFit: 'contain' }}
          autoPlay
          playsInline
          onClick={handlePlayPause}
        >
          Your browser does not support the video tag.
        </video>

        {/* Subtitle overlay - renders all subtitle types (text and bitmap/PGS) */}
        {/* Key changes on large seeks to force remount with fresh streamOffsetRef */}
        <SubtitleOverlay
          key={subtitleSeekKey}
          videoRef={videoRef}
          mediaId={mediaId}
          trackId={currentSubtitle?.id ?? null}
          isBitmap={currentSubtitle?.isBitmap}
          streamIndex={textStreamIndex}
          bitmapIndex={bitmapStreamIndex}
          streamOffsetRef={streamOffsetRef}
        />

        {/* Stats panel */}
        <StatsPanel
          stats={streamStats}
          networkStats={networkStats}
          isVisible={showDebugOverlay}
          onClose={() => setShowDebugOverlay(false)}
          isLoading={streamStatsLoading}
          selectedQuality={backendQualities.find(q => q.id === selectedQualityId) ?? null}
        />

        {/* Video controls */}
        <VideoControls
          videoRef={videoRef}
          isPlaying={isPlaying}
          currentTime={currentTime}
          duration={videoDuration}
          volume={volume}
          isMuted={isMuted}
          isFullscreen={isFullscreen}
          isPiP={isPiP}
          availableQualities={backendQualities}
          selectedQualityId={selectedQualityId}
          availableAudioTracks={availableAudioTracks}
          currentAudioTrack={currentAudioTrack}
          availableSubtitles={availableSubtitles}
          currentSubtitle={currentSubtitle?.id ?? null}
          playbackSpeed={playbackSpeed}
          metadata={metadata}
          showStats={showDebugOverlay}
          onPlayPause={handlePlayPause}
          onSeek={handleSeek}
          onVolumeChange={handleVolumeChangeControl}
          onMuteToggle={handleMuteToggle}
          onFullscreenToggle={handleFullscreenToggle}
          onPiPToggle={handlePiPToggle}
          onQualityChange={handleQualityChange}
          onAudioTrackChange={handleAudioTrackChange}
          onSubtitleChange={handleSubtitleChange}
          onSpeedChange={handleSpeedChange}
          onSkip={handleSkip}
          onToggleStats={() => setShowDebugOverlay((prev) => !prev)}
          onPlayNext={handlePlayNext}
          onPlayPrev={onPlayPrev}
        />

        {/* Auto-play countdown overlay - bottom-right card */}
        {autoPlayCountdown !== null && onAutoPlayNext && (
          <div className="absolute bottom-20 right-4 z-20">
            <div className="bg-gray-900/95 backdrop-blur-md rounded-xl p-4 shadow-2xl border border-white/10 min-w-[220px] max-w-[280px]">
              <p className="text-white/70 text-xs font-medium uppercase tracking-wider mb-1">
                Next episode
              </p>
              {nextEpisodeInfo && (
                <p className="text-white text-sm font-medium mb-2 line-clamp-1">
                  S{nextEpisodeInfo.season}:E{nextEpisodeInfo.episode}
                  {nextEpisodeInfo.episodeTitle ? ` - ${nextEpisodeInfo.episodeTitle}` : ''}
                </p>
              )}
              <div className="flex items-center justify-between gap-3">
                <p className="text-primary-400 font-bold text-lg tabular-nums">{autoPlayCountdown}s</p>
                <div className="flex gap-2">
                  <button
                    id="play-now-btn"
                    onClick={handlePlayNow}
                    className="px-3 py-1.5 text-xs bg-primary-600 hover:bg-primary-500 text-white rounded-lg transition-colors cursor-pointer font-medium whitespace-nowrap"
                  >
                    Play Now
                  </button>
                  <button
                    onClick={handleCancelAutoPlay}
                    className="px-3 py-1.5 text-xs bg-white/10 hover:bg-white/20 text-white rounded-lg transition-colors cursor-pointer font-medium whitespace-nowrap"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}