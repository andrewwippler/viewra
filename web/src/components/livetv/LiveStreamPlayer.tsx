/**
 * LiveStreamPlayer Component
 *
 * Lightweight video player for live TV streams with HLS support.
 * Designed for real-time streaming with DVR pause/resume functionality.
 */

import { useEffect, useRef, useState, useCallback } from 'react'
import Hls from 'hls.js'
import { getAuthHeaders } from '@/lib/utils/authFetch'
import { logger } from '@/lib/utils/logger'
import { ensureVideoUnmuted } from '@/lib/utils/videoUtils'
import { ArrowLeft, Volume2, VolumeX, Maximize, Minimize, Loader, Pause, Play, Square } from 'lucide-react'
import type { LiveChannel, EpgProgram } from '@/lib/api/livetv'

interface LiveStreamPlayerProps {
  channel: LiveChannel
  currentProgram?: EpgProgram
  libraryId?: number
  onClose?: () => void
  onError?: (error: string | null) => void
  onDVRLimitReached?: () => void
}

// HLS configuration optimized for low-latency live streaming with DVR
const LIVE_HLS_CONFIG = {
  MAX_BUFFER_LENGTH: 3,
  MAX_MAX_BUFFER_LENGTH: 10,
  MAX_BUFFER_SIZE: 20 * 1000 * 1000,
  MAX_BUFFER_HOLE: 0.3,
  LOW_LATENCY_MODE: true,
  BACK_BUFFER_LENGTH: 10,
  HIGH_BUFFER_WATCHDOG_PERIOD: 1,
  FRAG_LOADING_MAX_RETRY: 5,
  FRAG_LOADING_MAX_RETRY_TIMEOUT: 32000,
  NUDGE_OFFSET: 0.1,
  NUDGE_MAX_RETRY: 5,
}

const ENABLE_WORKER = import.meta.env.VITE_UI_MODE !== 'tv'

const formatTime = (seconds: number): string => {
  if (!isFinite(seconds) || seconds < 0) {return '0:00'}
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m}:${s.toString().padStart(2, '0')}`
}

interface DVRStatus {
  isPaused: boolean
  position: number
  retainedSegments: number
}

export const LiveStreamPlayer = ({
  channel,
  currentProgram,
  libraryId,
  onClose,
  onError,
  onDVRLimitReached,
}: LiveStreamPlayerProps) => {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const [isPlaying, setIsPlaying] = useState(false)
  const [isBuffering, setIsBuffering] = useState(true)
  const [isMuted, setIsMuted] = useState(false)
  const [volume, setVolume] = useState(1)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showControls, setShowControls] = useState(true)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [dvrStatus, setDvrStatus] = useState<DVRStatus>({ isPaused: false, position: 0, retainedSegments: 0 })

  const controlsTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const localErrorRef = useRef<string | null>(null)
  const currentTimeRef = useRef<number>(0)
  const seekingRef = useRef(false)

  // Handle error reporting
  const handleError = useCallback((err: string | null) => {
    localErrorRef.current = err
    setError(err)
    if (onError) {
      onError(err)
    }
  }, [onError])

  // Get precise position from current HLS segment
  const getPrecisePosition = useCallback((): number => {
    const video = videoRef.current
    return video ? video.currentTime : 0
  }, [])

  const handleSeek = useCallback((time: number) => {
    const video = videoRef.current
    if (!video) {return}
    seekingRef.current = true
    video.currentTime = time
    setCurrentTime(time)
    seekingRef.current = false
  }, [])

  // Pause live stream (enables DVR)
  const pauseStream = useCallback(async () => {
    if (!libraryId || dvrStatus.isPaused) {return}

    const position = getPrecisePosition()

    try {
      const headers = await getAuthHeaders()
      const res = await fetch(`/api/livetv/${libraryId}/channels/${channel.id}/pause`, {
        method: 'POST',
        headers: {
          ...headers,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ position }),
      })

      if (res.status === 409) {
        handleError('Another channel is currently paused')
        return
      }

      if (res.status === 410) {
        handleError('DVR limit reached')
        setDvrStatus((prev) => ({ ...prev, isPaused: true }))
        if (onDVRLimitReached) {
          onDVRLimitReached()
        }
        return
      }

      if (!res.ok) {
        throw new Error('Failed to pause stream')
      }

      const data = await res.json()
      setDvrStatus({
        isPaused: true,
        position: data.position,
        retainedSegments: data.retained_segments,
      })

      hlsRef.current?.stopLoad()
      videoRef.current?.pause()
      setIsPlaying(false)
    } catch (e: unknown) {
      handleError(e instanceof Error ? e.message : 'Failed to pause stream')
    }
  }, [libraryId, channel.id, dvrStatus.isPaused, getPrecisePosition, handleError, onDVRLimitReached])

  // Resume live stream
  const resumeStream = useCallback(async () => {
    if (!libraryId || !dvrStatus.isPaused) {return}

    try {
      const headers = await getAuthHeaders()
      const res = await fetch(`/api/livetv/${libraryId}/channels/${channel.id}/resume`, {
        method: 'POST',
        headers,
      })

      if (!res.ok) {
        throw new Error('Failed to resume stream')
      }

      const data = await res.json()

      if (videoRef.current && data.position > 0) {
        videoRef.current.currentTime = data.position
      }

      setDvrStatus({
        isPaused: false,
        position: data.position,
        retainedSegments: 0,
      })

      hlsRef.current?.startLoad()
      videoRef.current?.play()
      setIsPlaying(true)
    } catch (e: unknown) {
      handleError(e instanceof Error ? e.message : 'Failed to resume stream')
    }
  }, [libraryId, channel.id, dvrStatus.isPaused, handleError])

  // Stop live stream
  const stopStream = useCallback(async () => {
    if (!libraryId) {return}

    try {
      const headers = await getAuthHeaders()
      await fetch(`/api/livetv/${libraryId}/channels/${channel.id}/stop`, {
        method: 'POST',
        headers,
      })
    } catch {
      // Ignore stop errors
    }

    if (onClose) {
      onClose()
    }
  }, [libraryId, channel.id, onClose])

  // Initialize HLS player
  useEffect(() => {
    const video = videoRef.current
    if (!video) {return}

    let streamUrl = channel.stream_url
    const isHlsStream = streamUrl.includes('.m3u8') || streamUrl.includes('m3u8')

    if (!isHlsStream && libraryId && channel.id) {
      streamUrl = `/api/livetv/${libraryId}/channels/${channel.id}/hls/playlist.m3u8`
    }

    const handlePlayError = (err: unknown) => {
      if (err instanceof DOMException) {
        if (err.name === 'NotAllowedError') {
          setIsPlaying(false)
        } else if (err.name === 'NotSupportedError') {
          handleError('This stream format is not supported by your browser')
        } else {
          handleError('Unable to play this stream')
        }
      } else {
        handleError('Playback failed')
      }
    }

    let destroyed = false
    let retryTimeout: ReturnType<typeof setTimeout> | null = null

    const handleNativeMetadata = () => {
      setIsBuffering(false)
      video.play().catch(handlePlayError)
    }

    const startPlayback = async () => {
      if (destroyed) {return}

      // Pre-resolve auth headers to fix async token mapping inside the sync HLS.js instantiation
      let resolvedToken: string | null = null
      try {
        const authHeaders = await getAuthHeaders()
        if (authHeaders?.['Authorization']) {
          resolvedToken = authHeaders['Authorization']
        }
      } catch (e) {
        logger.error('[LiveStream] Failed to gather auth headers for init context', e)
      }

      if (destroyed) {return}

      if (Hls.isSupported()) {
        const hls = new Hls({
          ...LIVE_HLS_CONFIG,
          enableWorker: ENABLE_WORKER,
          xhrSetup: (xhr) => {
            if (resolvedToken) {
              xhr.setRequestHeader('Authorization', resolvedToken)
            }
          },
        })

        hls.loadSource(streamUrl)

        let retryCount = 0
        const maxRetries = 15

        hls.on(Hls.Events.ERROR, (_event, data) => {
          if (data.fatal) {
            if (data.details === 'manifestLoadError' && retryCount < maxRetries) {
              retryCount++
              logger.warn('[LiveStream] Manifest load error, retrying', retryCount, '/', maxRetries)
              retryTimeout = setTimeout(() => {
                if (!destroyed) {
                  hls.destroy()
                  startPlayback()
                }
              }, 2000)
              return
            }

            logger.error('[LiveStream] HLS fatal error:', data.details, data.reason)
            handleError('Unable to load stream')
            hls.destroy()
          }
        })

        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          retryCount = 0
          setIsBuffering(false)
          video.muted = true
          video.play()
            .then(() => ensureVideoUnmuted(video))
            .catch(handlePlayError)
        })

        hls.on(Hls.Events.FRAG_LOADED, () => {
          setIsBuffering(false)
          handleError(null)
        })

        hls.attachMedia(video)
        hlsRef.current = hls
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        // Native fallback targeting Safari
        video.src = streamUrl
        video.addEventListener('loadedmetadata', handleNativeMetadata)
      } else {
        // Direct source edge processing
        video.src = streamUrl
        video.addEventListener('loadedmetadata', handleNativeMetadata)
      }
    }

    startPlayback()

    return () => {
      destroyed = true
      if (retryTimeout) {
        clearTimeout(retryTimeout)
      }
      if (video) {
        video.removeEventListener('loadedmetadata', handleNativeMetadata)
      }
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }
  }, [channel.stream_url, libraryId, channel.id, handleError])

  // Track playback state changes safely
  useEffect(() => {
    const video = videoRef.current
    if (!video) {return}

    const handlePlay = () => setIsPlaying(true)
    const handlePause = () => setIsPlaying(false)
    const handleWaiting = () => setIsBuffering(true)
    const handleCanPlay = () => setIsBuffering(false)
    const handleVolumeChange = () => {
      setVolume(video.volume)
      setIsMuted(video.muted)
    }
    const handleDurationChange = () => {
      if (isFinite(video.duration)) {
        setDuration(video.duration)
      }
    }
    const handleTimeUpdate = () => {
      currentTimeRef.current = video.currentTime
      if (!seekingRef.current) {
        setCurrentTime(video.currentTime)
      }
    }

    video.addEventListener('play', handlePlay)
    video.addEventListener('pause', handlePause)
    video.addEventListener('waiting', handleWaiting)
    video.addEventListener('canplay', handleCanPlay)
    video.addEventListener('volumechange', handleVolumeChange)
    video.addEventListener('timeupdate', handleTimeUpdate)
    video.addEventListener('durationchange', handleDurationChange)

    return () => {
      video.removeEventListener('play', handlePlay)
      video.removeEventListener('pause', handlePause)
      video.removeEventListener('waiting', handleWaiting)
      video.removeEventListener('canplay', handleCanPlay)
      video.removeEventListener('volumechange', handleVolumeChange)
      video.removeEventListener('timeupdate', handleTimeUpdate)
      video.removeEventListener('durationchange', handleDurationChange)
    }
  }, [])

  // Manage Fullscreen alterations
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement)
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange)
    }
  }, [])

  // Control overlay timer management
  const resetControlsTimer = useCallback(() => {
    setShowControls(true)
    if (controlsTimeoutRef.current) {
      clearTimeout(controlsTimeoutRef.current)
    }
    controlsTimeoutRef.current = setTimeout(() => {
      if (isPlaying) {
        setShowControls(false)
      }
    }, 3000)
  }, [isPlaying])

  useEffect(() => {
    return () => {
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current)
      }
    }
  }, [])

  const handlePlayPause = useCallback(() => {
    if (dvrStatus.isPaused) {
      resumeStream()
    } else if (isPlaying) {
      pauseStream()
    } else {
      videoRef.current?.play()
    }
  }, [isPlaying, dvrStatus.isPaused, pauseStream, resumeStream])

  const handleMuteToggle = useCallback(() => {
    const video = videoRef.current
    if (!video) {return}
    video.muted = !video.muted
  }, [])

  const handleVolumeChange = useCallback((newVolume: number) => {
    const video = videoRef.current
    if (!video) {return}
    video.volume = newVolume
    if (newVolume > 0 && video.muted) {
      video.muted = false
    }
  }, [])

  const handleFullscreenToggle = useCallback(() => {
    const container = containerRef.current
    if (!container) {return}
    if (document.fullscreenElement) {
      document.exitFullscreen()
    } else {
      container.requestFullscreen()
    }
  }, [])

  const dvrProgress = useCallback(() => {
    const maxSegments = 1800
    return Math.min((dvrStatus.retainedSegments / maxSegments) * 100, 100)
  }, [dvrStatus.retainedSegments])

  return (
    <div
      ref={containerRef}
      className="fixed inset-0 z-50 bg-black flex flex-col"
      onMouseMove={resetControlsTimer}
      onMouseLeave={() => isPlaying && setShowControls(false)}
    >
      {error && (
        <div className="absolute top-20 left-1/2 transform -translate-x-1/2 z-30 bg-red-600/90 text-white px-6 py-3 rounded-lg shadow-lg max-w-md text-center">
          <p className="text-sm font-medium">{error}</p>
        </div>
      )}

      <div className={`absolute top-0 left-0 right-0 z-30 flex items-center justify-between p-4 transition-opacity duration-300 ${showControls ? 'opacity-100' : 'opacity-0'}`}>
        {onClose && (
          <button
            onClick={onClose}
            className="flex items-center gap-2 px-3 py-2 bg-black/60 hover:bg-black/80 text-white rounded-lg transition-colors text-sm font-medium cursor-pointer"
          >
            <ArrowLeft className="w-4 h-4" />
            Back
          </button>
        )}
        {onClose && (
          <button
            onClick={onClose}
            className="px-4 py-2 bg-black/60 hover:bg-black/80 text-white rounded-lg transition-colors text-sm font-medium cursor-pointer"
          >
            Close
          </button>
        )}
      </div>

      {isBuffering && (
        <div className="absolute inset-0 z-20 flex items-center justify-center pointer-events-none">
          <Loader className="w-16 h-16 text-white animate-spin" />
        </div>
      )}

      <div
        className="flex-1 flex items-center justify-center bg-black relative cursor-pointer"
        onClick={handlePlayPause}
      >
        <video
          ref={videoRef}
          className="w-full h-full max-h-screen"
          style={{ objectFit: 'contain' }}
          playsInline
        >
          Your browser does not support video playback.
        </video>

        <div className="absolute top-4 left-4 flex items-center gap-2">
          <span className={`flex items-center gap-1.5 px-2 py-1 text-xs font-bold rounded uppercase tracking-wider ${
            dvrStatus.isPaused ? 'bg-amber-600 text-white' : 'bg-red-600 text-white'
          }`}>
            <span className="w-2 h-2 rounded-full animate-pulse bg-white" />
            {dvrStatus.isPaused ? 'DVR' : 'LIVE'}
          </span>
          <span className="px-2 py-1 bg-black/60 text-white text-xs font-mono rounded">
            {channel.channel_number}
          </span>
        </div>

        {currentProgram && showControls && (
          <div className="absolute bottom-16 left-4 right-4 md:left-auto md:right-4 md:w-80 bg-black/80 backdrop-blur-sm rounded-lg p-3 text-white">
            <div className="flex items-start gap-3">
              {channel.logo_url && (
                <img src={channel.logo_url} alt="" className="w-10 h-10 object-contain rounded shrink-0" />
              )}
              <div className="min-w-0 flex-1">
                <h3 className="font-semibold text-sm truncate">{channel.name}</h3>
                <p className="text-xs text-white/80 truncate mt-0.5">{currentProgram.title}</p>
                {currentProgram.description && (
                  <p className="text-xs text-white/60 line-clamp-2 mt-1 hidden md:block">
                    {currentProgram.description}
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        {dvrStatus.isPaused && (
          <div className="absolute bottom-32 left-4 right-4 md:left-4 md:right-auto md:w-64">
            <div className="bg-black/60 backdrop-blur-sm rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-white/70">DVR Buffer</span>
                <span className="text-xs text-white/70">{dvrStatus.retainedSegments} / 1800</span>
              </div>
              <div className="h-1.5 bg-white/20 rounded-full overflow-hidden">
                <div
                  className="h-full bg-amber-500 rounded-full transition-all"
                  style={{ width: `${dvrProgress()}%` }}
                />
              </div>
            </div>
          </div>
        )}

        <div className={`absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent pt-8 pb-4 px-4 transition-opacity duration-300 ${showControls ? 'opacity-100' : 'opacity-0'}`}>
          <div className="mb-3">
            <div className="relative h-1 bg-white/20 rounded-full group cursor-pointer"
              onClick={(e) => {
                e.stopPropagation()
                const rect = e.currentTarget.getBoundingClientRect()
                const pos = (e.clientX - rect.left) / rect.width
                const seekTime = pos * (duration || currentTime * 2 || 120)
                handleSeek(seekTime)
              }}
            >
              <div
                className="absolute top-0 left-0 h-full bg-red-500 rounded-full"
                style={{ width: `${duration > 0 ? (currentTime / duration) * 100 : 0}%` }}
              />
            </div>
            <div className="flex justify-between mt-1">
              <span className="text-xs text-white/70 font-mono">{formatTime(currentTime)}</span>
              <span className="text-xs text-white/70 font-mono">
                {dvrStatus.isPaused ? formatTime(dvrStatus.position) : 'LIVE'}
              </span>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  handlePlayPause()
                }}
                className="p-2 hover:bg-white/20 rounded-lg transition-colors cursor-pointer text-white"
                title={dvrStatus.isPaused ? 'Resume' : isPlaying ? 'Pause (DVR)' : 'Play'}
              >
                {dvrStatus.isPaused ? (
                  <Play className="w-5 h-5" fill="currentColor" />
                ) : isPlaying ? (
                  <Pause className="w-5 h-5" />
                ) : (
                  <Play className="w-5 h-5" fill="currentColor" />
                )}
              </button>

              <button
                onClick={(e) => {
                  e.stopPropagation()
                  stopStream()
                }}
                className="p-2 hover:bg-white/20 rounded-lg transition-colors cursor-pointer text-white"
                title="Stop"
              >
                <Square className="w-4 h-4" fill="currentColor" />
              </button>

              <div className="w-px h-6 bg-white/20 mx-1" />

              <button
                onClick={(e) => {
                  e.stopPropagation()
                  handleMuteToggle()
                }}
                className="p-2 hover:bg-white/20 rounded-lg transition-colors cursor-pointer text-white"
                title={isMuted ? 'Unmute' : 'Mute'}
              >
                {isMuted || volume === 0 ? (
                  <VolumeX className="w-5 h-5" />
                ) : (
                  <Volume2 className="w-5 h-5" />
                )}
              </button>
              <input
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={isMuted ? 0 : volume}
                onChange={(e) => {
                  e.stopPropagation()
                  handleVolumeChange(parseFloat(e.target.value))
                }}
                className="w-20 h-1 bg-white/30 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:rounded-full"
              />
            </div>

            <div className="hidden md:flex items-center gap-3">
              <div className="text-right">
                <p className="text-sm font-medium text-white truncate max-w-[200px]">{channel.name}</p>
                {currentProgram && (
                  <p className="text-xs text-white/70 truncate max-w-[200px]">{currentProgram.title}</p>
                )}
              </div>
              {channel.logo_url && (
                <img src={channel.logo_url} alt="" className="w-8 h-8 object-contain rounded" />
              )}
            </div>

            <button
              onClick={(e) => {
                e.stopPropagation()
                handleFullscreenToggle()
              }}
              className="p-2 hover:bg-white/20 rounded-lg transition-colors cursor-pointer text-white"
              title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
            >
              {isFullscreen ? (
                <Minimize className="w-5 h-5" />
              ) : (
                <Maximize className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default LiveStreamPlayer