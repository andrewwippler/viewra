/**
 * TV Season Detail Page
 * * TV-friendly interface showing episodes as clickable cards with thumbnails.
 * Navigation: Uses linear tab-order navigation (matches jellyfin-webos).
 */

import { useEffect, useRef, useState, useMemo } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, ChevronLeft, ChevronRight } from 'lucide-react'
import { tvApi } from '@/lib/api/tv'
import { useMediaPlayback, useWebOSInputNavigation, useWatchedList } from '@/lib/hooks'
import { VideoPlayerContainer } from '@/components/media'
import { MediaPoster } from '@/components/media/MediaPoster'
import { WatchedBadge } from '@/components/media/WatchedBadge'

import { cn } from '@/lib/utils'
import type { TVEpisodeResponse } from '@/lib/types/tv'

const chunk = <T,>(arr: T[], size: number): T[][] =>
  Array.from({ length: Math.ceil(arr.length / size) }, (_, i) =>
    arr.slice(i * size, i * size + size)
  )

interface TvSeasonDetailProps {
  showId: number
  seasonNumber: number
  onBack?: () => void
  episodeId?: number
  initialTime?: number
}

export const TvSeasonDetail = ({ showId, seasonNumber, onBack, episodeId: urlEpisodeId, initialTime: urlTimePosition }: TvSeasonDetailProps) => {
  const navigate = useNavigate()
  const { playMedia, playbackState, stopPlayback, changeQuality } = useMediaPlayback()

  // Enable row-based navigation for WebOS
  useWebOSInputNavigation({
    enabled: !playbackState.isPlaying,
    rowSelector: '[data-row]',
    onBackPress: () => {
      if (onBack) {
        onBack()
      } else {
        navigate({ to: `/webos/tv/${showId}` })
      }
    },
  })

  const [showAutoPlayEnded, setShowAutoPlayEnded] = useState(false)
  const isClosingRef = useRef(false)
  const prevPlayingRef = useRef(playbackState.isPlaying)

  const {
    data: showData,
    isLoading: isLoadingShow,
    error: showError,
  } = useQuery({
    queryKey: ['tv-show', showId],
    queryFn: () => tvApi.getShow(showId),
  })

  const {
    data: episodesData,
    isLoading: isLoadingEpisodes,
    error: episodesError,
  } = useQuery({
    queryKey: ['tv-episodes', showId],
    queryFn: () => tvApi.listEpisodesByShowId(showId),
  })

  const { data: watchedData } = useWatchedList({ limit: 10000 })
  const watchedEpisodeIds = useMemo(
    () => new Set(watchedData?.progress?.map((p) => p.media_id) || []),
    [watchedData],
  )

  const show = showData?.data && 'title' in showData.data ? showData.data : null
  const isLoading = isLoadingShow || isLoadingEpisodes
  const error = showError || episodesError

  const seasonEpisodes = useMemo(() => {
    if (!episodesData?.data || !('episodes' in episodesData.data)) { return [] }
    return (episodesData.data.episodes || [])
      .filter((ep: TVEpisodeResponse) => ep.season === seasonNumber)
      .sort((a: TVEpisodeResponse, b: TVEpisodeResponse) => (a.episode ?? 0) - (b.episode ?? 0))
  }, [episodesData, seasonNumber])

  const showTitle = show?.title || ''

  const playingEpisode = useMemo(() => {
    const episode = seasonEpisodes.find((ep) => ep.id === playbackState.mediaId)
    if (!episode) { return undefined }
    return {
      ...episode,
      show_title: showTitle,
      show_id: showId,
    }
  }, [seasonEpisodes, playbackState.mediaId, showTitle, showId])

  // Base chronological sorting structure
  const allSortedEpisodes = useMemo(() => {
    if (!episodesData?.data || !('episodes' in episodesData.data)) { return [] }
    return [...(episodesData.data.episodes || [])].sort((a, b) => {
      const aSeason = a.season ?? 0
      const bSeason = b.season ?? 0
      if (aSeason === 0 && bSeason !== 0) { return 1 }
      if (bSeason === 0 && aSeason !== 0) { return -1 }
      if (aSeason !== bSeason) { return aSeason - bSeason}
      return (a.episode ?? 0) - (b.episode ?? 0)
    })
  }, [episodesData])

  const currentIndex = useMemo(() => {
    if (!playingEpisode) {return -1}
    return allSortedEpisodes.findIndex((ep) => ep.id === playingEpisode.id)
  }, [playingEpisode, allSortedEpisodes])

  /**
   * FIXED PLAY NEXT FEATURE
   * Mimics the backend GetNextEpisodeUseCase by evaluating watch progress history.
   * Looks ahead for the next unwatched episode. If all subsequent ones are watched,
   * falls back to the immediate next chronological item.
   */
  const nextEpisode = useMemo(() => {
    if (currentIndex === -1 || currentIndex === allSortedEpisodes.length - 1) {return null}

    // Look ahead for the first unwatched episode remaining in the collection
    const remainingEpisodes = allSortedEpisodes.slice(currentIndex + 1)
    const nextUnwatched = remainingEpisodes.find((ep) => ep.id && !watchedEpisodeIds.has(ep.id))

    // Fall back to immediate next chronological episode if no remaining unwatched episodes exist
    return nextUnwatched || remainingEpisodes[0] || null
  }, [currentIndex, allSortedEpisodes, watchedEpisodeIds])

  const nextEpisodeRef = useRef(nextEpisode)
  nextEpisodeRef.current = nextEpisode

  // Previous episode calculation stays linear/chronological for user convenience
  const prevEpisode = useMemo(() => {
    if (currentIndex <= 0) {return null}
    return allSortedEpisodes[currentIndex - 1]
  }, [currentIndex, allSortedEpisodes])

  const handlePlayEpisode = async (episode: TVEpisodeResponse, startTime?: number) => {
    if (!episode.id) { return }

    navigate({
      to: `/webos/tv/${showId}/season/${episode.season ?? seasonNumber}`,
      search: (prev) => ({
        ...prev,
        episodeId: episode.id,
        t: startTime && startTime > 0 ? Math.floor(startTime) : undefined,
      }),
    })

    await playMedia(episode.id, episode, startTime)
  }

  const handlePlayNextEpisode = async () => {
    if (nextEpisode) {
      setShowAutoPlayEnded(false)
      await handlePlayEpisode(nextEpisode)
    }
  }

  const handlePlayPrevEpisode = async () => {
    if (prevEpisode) {
      setShowAutoPlayEnded(false)
      await handlePlayEpisode(prevEpisode)
    }
  }

  const handleClosePlayer = () => {
    isClosingRef.current = true
    setShowAutoPlayEnded(false)
    stopPlayback()
    if (urlEpisodeId) {
      navigate({
        to: `/webos/tv/${showId}/season/${seasonNumber}`,
        search: (prev) => ({ ...prev, episodeId: undefined, t: undefined }),
      })
    }
  }

  const handleTimeUpdate = (time: number) => {
    if (urlEpisodeId && time > 0) {
      navigate({
        to: `/webos/tv/${showId}/season/${seasonNumber}`,
        search: (prev) => ({
          ...prev,
          t: Math.floor(time),
        }),
        replace: true,
      })
    }
  }

  // Auto-play hook execution for deep links
  useEffect(() => {
    if (isClosingRef.current) { return }
    if (urlEpisodeId && !playbackState.isPlaying && !playbackState.mediaId && seasonEpisodes.length > 0) {
      const episode = seasonEpisodes.find((ep) => ep.id === urlEpisodeId)
      if (episode) {
        handlePlayEpisode(episode, urlTimePosition)
      }
    }
  }, [urlEpisodeId, seasonEpisodes, playbackState.isPlaying, playbackState.mediaId])

  useEffect(() => {
    if (!urlEpisodeId) {
      isClosingRef.current = false
    }
  }, [urlEpisodeId])

  // Monitor playback termination states
  useEffect(() => {
    if (prevPlayingRef.current && !playbackState.isPlaying && !isClosingRef.current) {
      if (nextEpisodeRef.current) {
        setShowAutoPlayEnded(true)
      } else {
        navigate({ to: '/webos' })
      }
    }
    prevPlayingRef.current = playbackState.isPlaying
  }, [playbackState.isPlaying, navigate])

  if ((playbackState.isPlaying || showAutoPlayEnded) && playingEpisode) {
    return (
      <VideoPlayerContainer
        playbackState={playbackState}
        media={playingEpisode}
        onClose={handleClosePlayer}
        onTimeUpdate={handleTimeUpdate}
        onQualityChange={changeQuality}
        showOnEnd={showAutoPlayEnded}
        nextEpisodeInfo={nextEpisode ? {
          title: nextEpisode.title || nextEpisode.episode_title || '',
          season: nextEpisode.season ?? 0,
          episode: nextEpisode.episode ?? 0,
          episodeTitle: nextEpisode.episode_title,
        } : undefined}
        onAutoPlayNext={handlePlayNextEpisode}
        onAutoPlayCancel={() => setShowAutoPlayEnded(false)}
        onPlayNext={nextEpisode ? handlePlayNextEpisode : undefined}
        onPlayPrev={prevEpisode ? handlePlayPrevEpisode : undefined}
      />
    )
  }

  const handleBackClick = () => {
    if (onBack) {
      onBack()
    } else {
      navigate({ to: `/webos/tv/${showId}` })
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-neutral-900 flex items-center justify-center">
        <div className="animate-pulse text-neutral-400 text-2xl">Loading...</div>
      </div>
    )
  }

  if (error || !show) {
    return (
      <div className="min-h-screen bg-neutral-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-error-400 text-2xl mb-4">{error ? String(error) : 'Show not found'}</p>
          <button onClick={handleBackClick} className="px-6 py-3 bg-neutral-800 rounded-lg text-white">
            Go Back
          </button>
        </div>
      </div>
    )
  }

  const seasonLabel = seasonNumber === 0 ? 'Specials' : `Season ${seasonNumber}`

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-900 via-neutral-900 to-neutral-950 text-white relative overflow-y-auto">
      <div data-row className="p-4 flex items-center gap-4 sticky top-0 bg-neutral-900/95 z-10">
        <button
          onClick={handleBackClick}
          className="p-3 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50 transition-colors"
        >
          <ArrowLeft className="w-7 h-7" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold truncate">{showTitle}</h1>
          <p className="text-base text-neutral-400">{seasonLabel}</p>
        </div>

        <div className="flex gap-2">
          {seasonNumber > 0 && (
            <button
              onClick={() => navigate({ to: `/webos/tv/${showId}/season/${seasonNumber - 1}` })}
              className="p-2 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50"
            >
              <ChevronLeft className="w-6 h-6" />
            </button>
          )}
          {seasonEpisodes.length > 0 && (
            <button
              onClick={() => navigate({ to: `/webos/tv/${showId}/season/${seasonNumber + 1}` })}
              className="p-2 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50"
            >
              <ChevronRight className="w-6 h-6" />
            </button>
          )}
        </div>
      </div>

      <div className="px-4 pb-24">
        <h3 className="text-xl font-semibold mb-4">{seasonEpisodes.length} Episodes</h3>

        {chunk(seasonEpisodes, 5).map((group, rowIndex) => (
          <div data-row key={rowIndex} className="grid grid-cols-5 gap-2 mb-2">
            {group.map((episode: TVEpisodeResponse) => {
              const episodeLabel = episode.episode !== undefined
                ? `S${episode.season}E${episode.episode}`
                : ''
              const episodeTitle = episode.episode_title || episode.title || 'Unknown Episode'

              return (
                <button
                  key={episode.id}
                  id={rowIndex === 0 && group.indexOf(episode) === 0 ? 'connect' : undefined}
                  onClick={() => handlePlayEpisode(episode)}
                  className={cn(
                    'text-left rounded-lg overflow-hidden transition-all bg-neutral-800',
                    'focus:ring-4 focus:ring-primary-500/50'
                  )}
                >
                  <div className="w-full aspect-[2/3] bg-neutral-700 relative">
                    <MediaPoster
                      mediaId={episode.id ?? 0}
                      mediaType="tv-episode"
                      alt={episodeTitle}
                      preset="medium"
                      fallbackIcon="▶"
                      className="w-full h-full object-cover"
                    />
                    <div className="absolute top-2 left-2 px-2 py-1 bg-black/70 rounded text-sm font-bold text-white">
                      {episode.episode || rowIndex * 5 + group.indexOf(episode) + 1}
                    </div>
                    <WatchedBadge isWatched={watchedEpisodeIds.has(episode.id ?? 0)} />
                  </div>
                  <div className="p-2">
                    <p className="font-semibold text-white text-sm truncate">{episodeTitle}</p>
                    {episodeLabel && <p className="text-neutral-500 text-sm">{episodeLabel}</p>}
                  </div>
                </button>
              )
            })}
          </div>
        ))}
      </div>

      <div className="fixed bottom-0 left-0 right-0 p-4 text-center text-neutral-500 text-base bg-neutral-900/80">
        Use arrow keys to navigate • OK to play episode • Back to go back
      </div>
    </div>
  )
}