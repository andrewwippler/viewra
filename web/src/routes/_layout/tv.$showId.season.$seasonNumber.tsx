import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Card, CardContent, Button } from '@/components/ui'
import { EpisodeCard } from '@/components/tv'
import { VideoPlayerContainer } from '@/components/media'
import { PageHeader, LoadingPage, ErrorPage, EmptyState } from '@/components/common'
import { tvApi } from '@/lib/api/tv'
import { useMediaPlayback, BatchProgressProvider } from '@/lib/hooks'
import type { TVEpisodeResponse, TVShowDetailResponse } from '@/lib/types/tv'
import { logger } from '@/lib/utils/logger'

const SeasonDetail = () => {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { showId, seasonNumber } = Route.useParams()
  const search = Route.useSearch() as { episodeId?: number; t?: number }
  const urlEpisodeId = search.episodeId
  const urlTimePosition = search.t
  const showIdNumber = parseInt(showId, 10)

  const { playbackState, playMedia, stopPlayback, changeQuality } = useMediaPlayback()

  // Auto-play countdown state - keeps player visible after episode ends
  const [showAutoPlayEnded, setShowAutoPlayEnded] = useState(false)

  const {
    data: showData,
    isLoading: isLoadingShow,
    error: showError,
  } = useQuery({
    queryKey: ['tv-show', showIdNumber],
    queryFn: () => tvApi.getShow(showIdNumber),
  })

  const {
    data: episodesData,
    isLoading: isLoadingEpisodes,
    error: episodesError,
  } = useQuery({
    queryKey: ['tv-episodes', showIdNumber],
    queryFn: () => tvApi.listEpisodesByShowId(showIdNumber),
  })

  const allEpisodes = useMemo(() => {
    // Check if episodesData has the expected structure (not an error response)
    if (episodesData?.data && 'episodes' in episodesData.data) {
      return episodesData.data.episodes || []
    }
    return []
  }, [episodesData])
  const isLoading = isLoadingShow || isLoadingEpisodes
  const error = showError || episodesError
  const show = (showData?.data && 'title' in showData.data) ? showData.data as TVShowDetailResponse : null
  const showTitle = show?.title || ''

  // Filter episodes for this season and sort by episode number
  const seasonEpisodes = useMemo(() => {
    return allEpisodes
      .filter((ep: TVEpisodeResponse) => ep.season === parseInt(seasonNumber, 10))
      .sort((a: TVEpisodeResponse, b: TVEpisodeResponse) => (a.episode ?? 0) - (b.episode ?? 0))
  }, [allEpisodes, seasonNumber])

  // Find currently playing episode and enrich with show title and show_id
  const playingEpisode = useMemo(() => {
    const episode = seasonEpisodes.find((ep) => ep.id === playbackState.mediaId)
    if (!episode) {return undefined}
    // Enrich episode with show metadata for video player
    return {
      ...episode,
      show_title: showTitle,
      show_id: showIdNumber,
    }
  }, [seasonEpisodes, playbackState.mediaId, showTitle, showIdNumber])

  // Get next episode
  const nextEpisode = useMemo(() => {
    if (!playingEpisode) {return null}
    const currentIndex = seasonEpisodes.findIndex((ep) => ep.id === playingEpisode.id)
    if (currentIndex === -1 || currentIndex === seasonEpisodes.length - 1) {return null}
    return seasonEpisodes[currentIndex + 1]
  }, [playingEpisode, seasonEpisodes])

  // Get previous episode
  const prevEpisode = useMemo(() => {
    if (!playingEpisode) {return null}
    const currentIndex = seasonEpisodes.findIndex((ep) => ep.id === playingEpisode.id)
    if (currentIndex <= 0) {return null}
    return seasonEpisodes[currentIndex - 1]
  }, [playingEpisode, seasonEpisodes])

  // Ref to prevent auto-play from triggering during close
  const isClosingRef = useRef(false)

  // Auto-play episode if ID is in URL (only on initial load)
  useEffect(() => {
    // Don't auto-play if we're in the process of closing
    if (isClosingRef.current) {
      return
    }
    if (urlEpisodeId && !playbackState.isPlaying && !playbackState.mediaId && seasonEpisodes.length > 0) {
      const episode = seasonEpisodes.find((ep) => ep.id === urlEpisodeId)
      if (episode) {
        handlePlayEpisode(episode, urlTimePosition)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlEpisodeId, seasonEpisodes.length])

  // Reset closing flag when URL episode ID is cleared
  useEffect(() => {
    if (!urlEpisodeId) {
      isClosingRef.current = false
    }
  }, [urlEpisodeId])

  // Detect playback end to activate auto-play countdown
  const prevPlayingRef = useRef(playbackState.isPlaying)
  useEffect(() => {
    if (prevPlayingRef.current && !playbackState.isPlaying && nextEpisode && !isClosingRef.current) {
      setShowAutoPlayEnded(true)
    }
    prevPlayingRef.current = playbackState.isPlaying
  }, [playbackState.isPlaying, nextEpisode])

  const handlePlayEpisode = async (episode: TVEpisodeResponse, startTime?: number) => {
    logger.debug('Playing episode:', episode.show_title, `S${  episode.season  }E${  episode.episode}`)

    // Update URL with episode ID and optional time position
    navigate({
      to: `/tv/${showId}/season/${seasonNumber}`,
      search: {
        episodeId: episode.id,
        t: startTime && startTime > 0 ? Math.floor(startTime) : undefined
      }
    })

    // Trigger playback, passing URL time if available
    await playMedia(episode.id ?? 0, episode, startTime)
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

  // Handle time position updates from video player
  const handleTimeUpdate = (time: number) => {
    if (urlEpisodeId && time > 0) {
      navigate({
        to: `/tv/${showId}/season/${seasonNumber}`,
        search: {
          episodeId: urlEpisodeId,
          t: Math.floor(time)
        },
        replace: true, // Use replace to avoid polluting browser history
      })
    }
  }

  const handleClosePlayer = () => {
    // Set closing flag to prevent auto-play effect from re-triggering
    isClosingRef.current = true
    setShowAutoPlayEnded(false)
    stopPlayback()
    // Clear URL parameters if present
    if (urlEpisodeId) {
      navigate({
        to: `/tv/${showId}/season/${seasonNumber}`,
        search: { episodeId: undefined, t: undefined }
      })
    }
  }

  const handleBackClick = () => {
    navigate({ to: `/tv/${showId}` })
  }

  const videoPlayer = (
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

  if ((playbackState.isPlaying || showAutoPlayEnded) && playingEpisode) {
    return videoPlayer
  }

  if (isLoading) {
    return <LoadingPage text="Loading season..." />
  }

  if (error) {
    return <ErrorPage error={error} context="season" />
  }

  if (seasonEpisodes.length === 0) {
    return (
      <div className="p-8">
        <PageHeader
          title={`${showTitle} - Season ${seasonNumber}`}
          description="No episodes found"
          actions={<Button onClick={handleBackClick}>← Back to Show</Button>}
        />
        <Card>
          <CardContent>
            <EmptyState
              icon="📺"
              title="No episodes found"
              description="This season doesn't have any episodes in the selected library."
            />
          </CardContent>
        </Card>
      </div>
    )
  }

  const seasonLabel = parseInt(seasonNumber, 10) === 0 ? 'Specials' : `Season ${seasonNumber}`
  const episodeIds = seasonEpisodes.map((ep: TVEpisodeResponse) => ep.id ?? 0).filter((id): id is number => id !== 0)

  return (
    <div className="p-8">
      <PageHeader
        title={`${showTitle} - ${seasonLabel}`}
        description={`${seasonEpisodes.length} ${seasonEpisodes.length === 1 ? 'Episode' : 'Episodes'}`}
        actions={<Button onClick={handleBackClick}>← Back to Show</Button>}
      />

      {/* Show description */}
      {show?.plot && (
        <Card className="mb-6">
          <CardContent>
            <p className="text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed">
              {show.plot}
            </p>
            {show.genre && show.genre.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-3">
                {show.genre.map((g) => (
                  <span key={g} className="px-2 py-0.5 text-xs rounded-full bg-neutral-100 dark:bg-neutral-800 text-neutral-600 dark:text-neutral-400">
                    {g}
                  </span>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Episodes Grid */}
      <BatchProgressProvider mediaIds={episodeIds}>
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {seasonEpisodes.map((episode: TVEpisodeResponse) => (
            <EpisodeCard
              key={episode.id}
              episode={episode}
              onClick={() => handlePlayEpisode(episode)}
            />
          ))}
        </div>
      </BatchProgressProvider>
    </div>
  )
}

export const Route = createFileRoute('/_layout/tv/$showId/season/$seasonNumber')({
  component: SeasonDetail,
  validateSearch: (search: Record<string, unknown>) => {
    const episodeId = search.episodeId
    const parsedId = typeof episodeId === 'string' ? parseInt(episodeId, 10) : typeof episodeId === 'number' ? episodeId : undefined
    const t = search.t
    const parsedT = typeof t === 'string' ? parseInt(t, 10) : typeof t === 'number' ? t : undefined
    return {
      episodeId: parsedId && !isNaN(parsedId) ? parsedId : undefined,
      t: parsedT && !isNaN(parsedT) ? parsedT : undefined,
    }
  },
})
