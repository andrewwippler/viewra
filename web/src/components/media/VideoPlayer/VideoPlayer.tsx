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
import { useCallback, useEffect, useRef, useState, useMemo } from 'react'
import { VideoControls } from './VideoControls'
import { StatsPanel } from './StatsPanel'
import { SubtitleOverlay } from './SubtitleOverlay'
import type { VideoPlayerProps } from './VideoPlayer.types'
import { useGetApiMediaIdTracks } from '@/lib/api/generated/media/media'
import { getDeviceProfileHash } from '@/lib/capabilities'
import { getAutoFullscreenPreference, enterFullscreen, isInCSSFullscreen } from '@/utils/device'

declare const __TV_MODE__: boolean

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
  const [subtitleSeekKey, setSubtitleSeekKey] = useState(0)
  const [backendSessionId, setBackendSessionId] = useState<string | null>(null)

  // Auto-play conditional calculation based on backend layout specifications
  const isAutoplayEnabled = useMemo(() => {
    if (!onAutoPlayNext) {return false}
    // Checks backend setting preference fallback. Defaults to true if missing.
    return savedPreferences?.autoplay ?? true
  }, [onAutoPlayNext, savedPreferences?.autoplay])

  const [autoPlayCountdown, setAutoPlayCountdown] = useState<number | null>(null)
  const autoPlayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoPlayCountdownShownRef = useRef(false)
  const autoPlayTriggeredRef = useRef(false)

  const clearAutoPlayTimer = useCallback(() => {
    if (autoPlayTimerRef.current) {
      clearTimeout(autoPlayTimerRef.current)
      autoPlayTimerRef.current = null
    }
  }, [])

  const { data: tracksData } = useGetApiMediaIdTracks(mediaId)
  const subtitleTracksFromApi =
    tracksData?.status === 200 ? tracksData.data.subtitle_tracks || [] : []
  const audioTracksFromApi =
    tracksData?.status === 200 ? tracksData.data.audio_tracks || [] : []

  const [currentAudioStreamIndex, setCurrentAudioStreamIndex] = useState<number>(-1)
  const [trackSwitchPosition, setTrackSwitchPosition] = useState<number | null>(null)

  const buildEffectiveStreamUrl = () => {
    let url = streamUrl
    const params = new URLSearchParams()

    const urlParts = streamUrl.split('?')
    if (urlParts.length > 1) {
      const existingParams = new URLSearchParams(urlParts[1])
      existingParams.forEach((value, key) => params.set(key, value))
      url = urlParts[0]
    }

    if (currentAudioStreamIndex > 0) {
      params.set('audioTrack', String(currentAudioStreamIndex))
    }

    if (trackSwitchPosition !== null) {
      params.set('start', String(Math.floor(trackSwitchPosition)))
    }

    const paramStr = params.toString()
    return paramStr ? `${url}?${paramStr}` : url
  }

  const effectiveStreamUrl = buildEffectiveStreamUrl()

  useEffect(() => {
    if (trackSwitchPosition !== null && isPlaying) {
      setTrackSwitchPosition(null)
    }
  }, [effectiveStreamUrl, isPlaying, trackSwitchPosition])

  const effectiveInitialPosition = trackSwitchPosition ?? initialPosition

  const hasSavedSubtitlePref = savedPreferences?.selectedSubtitleTrack !== undefined
  const { availableSubtitles, currentSubtitle, setCurrentSubtitle, textStreamIndex, bitmapStreamIndex } = useSubtitles({
    subtitleTracks: subtitleTracksFromApi,
    preferredLanguage: 'off',
    preferSDH: false,
    preferForced: true,
    skipAutoSelect: hasSavedSubtitlePref,
  })

  const progressUpdater = useProgressUpdater(mediaId, videoDuration)

  const {
    hlsRef,
    streamOffsetRef,
  } = useHlsPlayer({
    videoRef,
    streamUrl: effectiveStreamUrl,
    initialPosition: effectiveInitialPosition,
    isHlsStream,
    onError: setError,
    onFragLoaded: (bytes, durationMs) => recordSample(bytes, durationMs),
    onSessionIdReceived: setBackendSessionId,
  })

  const availableAudioTracks = audioTracksFromApi
    .filter((track): track is typeof track & { stream_index: number } =>
      track.stream_index !== undefined
    )
    .map((track, index) => ({
      id: track.stream_index,
      name: track.title || `Track ${index + 1}`,
      language: track.language || 'Unknown',
    }))

  const currentAudioTrack = currentAudioStreamIndex > 0
    ? currentAudioStreamIndex
    : (availableAudioTracks[0]?.id ?? 0)

  const { recordSample, recordStall, networkStats } = useAutoQuality({
    enabled: isHlsStream,
    hlsInstance: hlsRef.current,
  })

  const { stats: streamStats, isLoading: streamStatsLoading, refresh: refreshStreamStats } = useStreamStats({
    mediaId,
    videoRef,
    hlsRef,
    networkStats,
    isPlaying,
    playbackMode: isHlsStream ? 'transcode' : 'direct',
    selectedQualityId,
    streamOffset: streamOffsetRef.current || 0,
    enabled: showDebugOverlay,
  })

  const {
    startSession,
    updateSessionId,
    endSession,
    recordQualitySwitch,
    recordStall: recordAnalyticsStall,
    recordPlayTime,
    recordStartupTime,
  } = usePlaybackAnalytics({ enabled: true })

  useEffect(() => {
    if (backendSessionId) {
      updateSessionId(backendSessionId)
    }
  }, [backendSessionId, updateSessionId])

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
      setSubtitleSeekKey((prev) => prev + 1)
    },
  })

  useVideoEvents({
    videoRef,
    mediaId,
    duration,
    videoDuration,
    isPlaying,
    streamOffsetRef,
    isSeekingRef,
    backendSessionId,
    onPlay: () => {
      setIsPlaying(true)
      autoPlayTriggeredRef.current = false
    },
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
      if (isAutoplayEnabled && !autoPlayCountdownShownRef.current && !autoPlayTriggeredRef.current) {
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

  useVideoKeyboard({
    videoRef,
    containerRef: videoContainerRef,
    videoDuration,
    onToggleDebug: () => setShowDebugOverlay((prev) => !prev),
  })

  const autoPlayNextRef = useRef(onAutoPlayNext)
  autoPlayNextRef.current = onAutoPlayNext
  const autoPlayCancelRef = useRef(onAutoPlayCancel)
  autoPlayCancelRef.current = onAutoPlayCancel

  useEffect(() => {
    if (typeof __TV_MODE__ === 'undefined' || !__TV_MODE__) {return}

    const handleKeyDown = (e: KeyboardEvent) => {
      const video = videoRef.current
      if (!video) {return}
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {return}

      switch (e.keyCode) {
        case 461:
          if (onClose) {
            e.preventDefault()
            e.stopPropagation()
            onClose()
          }
          return
        case 179:
        case 413:
        case 415:
          e.preventDefault()
          e.stopPropagation()
          if (video.paused) { video.play() } else { video.pause() }
          return
        case 412:
        case 464:
          e.preventDefault()
          e.stopPropagation()
          video.currentTime = Math.max(0, video.currentTime - 10)
          return
        case 417:
        case 465:
          e.preventDefault()
          e.stopPropagation()
          video.currentTime = Math.min(video.duration || videoDuration, video.currentTime + 10)
          return
        case 13:
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

  useEffect(() => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) {return}

    const video = videoRef.current
    if (!video) {return}

    if (metadata) {
      navigator.mediaSession.metadata = new MediaMetadata({
        title: metadata.title,
        artist: metadata.subtitle || undefined,
        artwork: metadata.posterUrl ? [{ src: metadata.posterUrl, sizes: '256x256', type: 'image/jpeg' }] : [],
      })
    }

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
  }, [metadata])

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
          selected_subtitle_track: currentSubtitle?.id ?? -1,
        })
        const apiUrl = `${window.location.origin}/api/progress`
        navigator.sendBeacon(apiUrl, new Blob([data], { type: 'application/json' }))
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [mediaId, currentTime, videoDuration, selectedQualityId, currentAudioStreamIndex, currentSubtitle])

  useEffect(() => {
    const video = videoRef.current
    const container = videoContainerRef.current
    if (!video || !container) {return}

    const autoFullscreenEnabled = getAutoFullscreenPreference()
    if (!autoFullscreenEnabled || !isPlaying) {return}

    let cancelled = false
    const timer = setTimeout(() => {
      if (cancelled) {return}
      if (document.fullscreenElement || isInCSSFullscreen(container)) {return}

      enterFullscreen(container).catch(() => {})
    }, 500)

    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [isPlaying])

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

  // Reset core tracking variables when media changing to clear stale playback bounds
  useEffect(() => {
    clearAutoPlayTimer()
    setCurrentTime(0)
    setAutoPlayCountdown(null)
    autoPlayCountdownShownRef.current = false
  }, [mediaId, clearAutoPlayTimer])

  // Time-based overlay presentation checks bounds matching user preferences
  const UP_NEXT_THRESHOLD = 20
  useEffect(() => {
    if (!isAutoplayEnabled) {return}
    if (autoPlayCountdownShownRef.current) {return}
    if (videoDuration <= 0 || currentTime <= 0) {return}
    const remaining = videoDuration - currentTime
    if (remaining > UP_NEXT_THRESHOLD || remaining <= 0) {return}

    autoPlayCountdownShownRef.current = true
    setAutoPlayCountdown(10)
  }, [currentTime, videoDuration, isAutoplayEnabled])

  useEffect(() => {
    if (autoPlayCountdown === null) {
      clearAutoPlayTimer()
      return
    }
    if (autoPlayCountdown <= 0) {
      clearAutoPlayTimer()
      autoPlayTriggeredRef.current = true
      autoPlayNextRef.current?.()
      setAutoPlayCountdown(null)
      return
    }
    const timer = setTimeout(() => {
      setAutoPlayCountdown((prev) => (prev ?? 0) - 1)
    }, 1000)
    autoPlayTimerRef.current = timer
    return () => clearTimeout(timer)
  }, [autoPlayCountdown, clearAutoPlayTimer])

  useEffect(() => {
    if (typeof __TV_MODE__ !== 'undefined' && __TV_MODE__ && autoPlayCountdown !== null) {
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

  const appliedAudioPrefRef = useRef(false)
  useEffect(() => {
    if (appliedAudioPrefRef.current) {return}
    if (!savedPreferences?.selectedAudioTrack) {return}
    if (availableAudioTracks.length === 0) {return}

    const trackExists = availableAudioTracks.some(t => t.id === savedPreferences.selectedAudioTrack)
    if (trackExists && savedPreferences.selectedAudioTrack !== currentAudioStreamIndex) {
      setCurrentAudioStreamIndex(savedPreferences.selectedAudioTrack)
    }
    appliedAudioPrefRef.current = true
  }, [savedPreferences, availableAudioTracks, currentAudioStreamIndex])

  const appliedSubtitlePrefRef = useRef(false)
  useEffect(() => {
    if (appliedSubtitlePrefRef.current) {return}
    if (savedPreferences?.selectedSubtitleTrack === undefined) {return}
    if (availableSubtitles.length === 0 && savedPreferences.selectedSubtitleTrack !== null) {return}

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

  useEffect(() => {
    appliedAudioPrefRef.current = false
    appliedSubtitlePrefRef.current = false
  }, [mediaId])

  useEffect(() => {
    progressUpdater.updatePreferences({
      selectedQuality: selectedQualityId,
      selectedAudioTrack: currentAudioStreamIndex > 0 ? currentAudioStreamIndex : null,
      selectedSubtitleTrack: currentSubtitle?.id ?? -1,
    })
  }, [selectedQualityId, currentAudioStreamIndex, currentSubtitle, progressUpdater])

  const handleQualityChange = useCallback(
    (qualityId: string) => {
      const video = videoRef.current
      if (!video || !onQualityChangeCallback) {return}

      const currentPosition = video.currentTime + (streamOffsetRef.current || 0)

      recordQualitySwitch(
        selectedQualityId,
        qualityId,
        'user_manual',
        currentPosition,
        null,
        null
      )

      onQualityChangeCallback(qualityId, currentPosition)

      setTimeout(() => refreshStreamStats(), 1000)
    },
    [onQualityChangeCallback, selectedQualityId, recordQualitySwitch, streamOffsetRef, refreshStreamStats]
  )

  const handleAudioTrackChange = useCallback(
    (streamIndex: number) => {
      const video = videoRef.current
      if (video) {
        const actualTime = video.currentTime + (streamOffsetRef.current || 0)
        setTrackSwitchPosition(actualTime)
      }
      setCurrentAudioStreamIndex(streamIndex)
    },
    [streamOffsetRef]
  )

  const handleSpeedChange = useCallback(
    (speed: number) => {
      handlePlaybackSpeedChange(speed)
      setPlaybackSpeed(speed)
    },
    [handlePlaybackSpeedChange]
  )

  const handleSubtitleChange = useCallback(
    (trackId: number | null) => {
      setCurrentSubtitle(trackId)
    },
    [setCurrentSubtitle]
  )

  return (
    <div className="fixed inset-0 z-50 bg-black flex flex-col">
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

      {error && (
        <div className="absolute top-20 left-1/2 transform -translate-x-1/2 z-20 bg-red-600/90 text-white px-6 py-3 rounded-lg shadow-lg max-w-md">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

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

        <StatsPanel
          stats={streamStats}
          networkStats={networkStats}
          isVisible={showDebugOverlay}
          onClose={() => setShowDebugOverlay(false)}
          isLoading={streamStatsLoading}
          selectedQuality={backendQualities.find(q => q.id === selectedQualityId) ?? null}
        />

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

        {autoPlayCountdown !== null && isAutoplayEnabled && (
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