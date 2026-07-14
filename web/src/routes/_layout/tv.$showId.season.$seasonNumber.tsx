import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Card, CardContent, Button } from '@/components/ui'
import { EpisodeCard } from '@/components/tv'
import { VideoPlayerContainer, MarkWatchedButton } from '@/components/media'
import { PageHeader, LoadingPage, ErrorPage, EmptyState } from '@/components/common'
import { tvApi } from '@/lib/api/tv'
import { useMediaPlayback, BatchProgressProvider, useBatchProgress } from '@/lib/hooks'
import type { TVEpisodeResponse, TVShowDetailResponse } from '@/lib/types/tv'
import { logger } from '@/lib/utils/logger'
import { AdminActions } from '@/features/nitpicky-edits'

// Simple interface fallback assuming useBatchProgress structure
interface BatchProgressContextType {
  progressRecord?: Record<number, { is_watched: boolean }>
}

const EpisodeAdminRow = ({ episodeId }: { episodeId: number }) => {
  const { progress } = useBatchProgress(episodeId)
  return (
    <div className="flex justify-end">
      <MarkWatchedButton
        mediaId={episodeId}
        isWatched={progress?.is_watched ?? false}
        variant="compact"
      />
    </div>
  )
}

const SeasonDetail = () => {
  const navigate = useNavigate()
  const { showId, seasonNumber } = Route.useParams()
  const search = Route.useSearch()
  const urlEpisodeId = search.episodeId
  const urlTimePosition = search.t
  const showIdNumber = parseInt(showId, 10)

  const { playbackState, playMedia, stopPlayback, changeQuality } = useMediaPlayback()
  const [showAutoPlayEnded, setShowAutoPlayEnded] = useState(false)

  // Context hook placeholder: Adjust if your project accesses global batch state differently
  // If useBatchProgress only works per item, you can fetch `useBatchProgressContext()` if available.
  // Below we default to smart chronological arrays but group them safely.

  const { data: showData, isLoading: isLoadingShow, error: showError } = useQuery({
    queryKey: ['tv-show', showIdNumber],
    queryFn: () => tvApi.getShow(showIdNumber),
  })

  const { data: episodesData, isLoading: isLoadingEpisodes, error: episodesError } = useQuery({
    queryKey: ['tv-episodes', showIdNumber],
    queryFn: () => tvApi.listEpisodesByShowId(showIdNumber),
  })

  const allEpisodes = useMemo(() => {
    if (episodesData?.data && 'episodes' in episodesData.data) {
      return episodesData.data.episodes || []
    }
    return []
  }, [episodesData])

  const isLoading = isLoadingShow || isLoadingEpisodes
  const error = showError || episodesError
  const show = (showData?.data && 'title' in showData.data) ? showData.data as TVShowDetailResponse : null
  const showTitle = show?.title || ''

  const seasonEpisodes = useMemo(() => {
    return allEpisodes
      .filter((ep: TVEpisodeResponse) => ep.season === parseInt(seasonNumber, 10))
      .sort((a: TVEpisodeResponse, b: TVEpisodeResponse) => (a.episode ?? 0) - (b.episode ?? 0))
  }, [allEpisodes, seasonNumber])

  const seasonDbId = useMemo(() => seasonEpisodes[0]?.season_id, [seasonEpisodes])

  const playingEpisode = useMemo(() => {
    const episode = seasonEpisodes.find((ep) => ep.id === playbackState.mediaId)
    if (!episode) return undefined
    return { ...episode, show_title: showTitle, show_id: showIdNumber }
  }, [seasonEpisodes, playbackState.mediaId, showTitle, showIdNumber])

  // Chronological foundation sorting
  const allSortedEpisodes = useMemo(() => {
    return [...allEpisodes].sort((a, b) => {
      const aSeason = a.season ?? 0
      const bSeason = b.season ?? 0
      if (aSeason === 0 && bSeason !== 0) return 1
      if (bSeason === 0 && aSeason !== 0) return -1
      if (aSeason !== bSeason) return aSeason - bSeason
      return (a.episode ?? 0) - (b.episode ?? 0)
    })
  }, [allEpisodes])

  /**
   * FIXING THE PLAY NEXT BUG
   * Realistically, true user progress requires calling your backend endpoint:
   * GET /api/tv/shows/:id/next-episode
   * Alternatively, we can calculate the absolute fallback index below.
   */
  const currentEpisodeIndex = useMemo(() => {
    if (!playingEpisode) return -1
    return allSortedEpisodes.findIndex((ep) => ep.id === playingEpisode.id)
  }, [playingEpisode, allSortedEpisodes])

  const nextEpisode = useMemo(() => {
    if (currentEpisodeIndex === -1 || currentEpisodeIndex === allSortedEpisodes.length - 1) return null
    return allSortedEpisodes[currentEpisodeIndex + 1]
  }, [currentEpisodeIndex, allSortedEpisodes])

  const prevEpisode = useMemo(() => {
    if (currentEpisodeIndex <= 0) return null
    return allSortedEpisodes[currentEpisodeIndex - 1]
  }, [currentEpisodeIndex, allSortedEpisodes])

  const isClosingRef = useRef(false)

  // Handle Initial Boot URL Deep-linking
  useEffect(() => {
    if (isClosingRef.current) return

    if (urlEpisodeId && !playbackState.isPlaying && seasonEpisodes.length > 0) {
      const episode = seasonEpisodes.find((ep) => ep.id === urlEpisodeId)
      if (episode && playbackState.mediaId !== episode.id) {
        handlePlayEpisode(episode, urlTimePosition)
      }
    }
  }, [urlEpisodeId, seasonEpisodes, playbackState.isPlaying, playbackState.mediaId])

  useEffect(() => {
    if (!urlEpisodeId) {
      isClosingRef.current = false
    }
  }, [urlEpisodeId])

  // Watch for active context changes to bring up Autoplay Prompt cards
  const prevPlayingRef = useRef(playbackState.isPlaying)
  useEffect(() => {
    if (prevPlayingRef.current && !playbackState.isPlaying && nextEpisode && !isClosingRef.current) {
      setShowAutoPlayEnded(true)
    }
    prevPlayingRef.current = playbackState.isPlaying
  }, [playbackState.isPlaying, nextEpisode])

  const handlePlayEpisode = async (episode: TVEpisodeResponse, startTime?: number) => {
    logger.debug('Playing episode:', episode.show_title, `S${episode.season}E${episode.episode}`)

    navigate({
      to: `/tv/${showId}/season/${episode.season ?? seasonNumber}`,
      search: {
        episodeId: episode.id,
        t: startTime && startTime > 0 ? Math.floor(startTime) : undefined
      }
    })

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

  const handleTimeUpdate = (time: number) => {
    if (urlEpisodeId && time > 0) {
      navigate({
        to: `/tv/${showId}/season/${seasonNumber}`,
        search: (prev) => ({ ...prev, t: Math.floor(time) }),
        replace: true,
      })
    }
  }

  const handleClosePlayer = () => {
    isClosingRef.current = true
    setShowAutoPlayEnded(false)
    stopPlayback()
    if (urlEpisodeId) {
      navigate({
        to: `/tv/${showId}/season/${seasonNumber}`,
        search: (prev) => ({ ...prev, episodeId: undefined, t: undefined })
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

  if (isLoading) return <LoadingPage text="Loading season..." />
  if (error) return <ErrorPage error={error} context="season" />

  if (seasonEpisodes.length === 0) {
    return (
      <div className="p-8">
        <PageHeader
          title={`${showTitle} - Season ${seasonNumber}`}
          description="No episodes found"
          actions={
            <div className="flex items-center gap-2">
              <AdminActions
                mediaType="tv"
                mediaId={showIdNumber}
                mediaTitle={show?.title || ''}
                seasonId={seasonDbId}
                onDeleteNavigate="/tv"
              />
              <Button onClick={handleBackClick}>← Back to Show</Button>
            </div>
          }
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
  const episodeIds = seasonEpisodes.map((ep) => ep.id ?? 0).filter((id): id is number => id !== 0)

  return (
    <div className="p-8">
      <PageHeader
        title={`${showTitle} - ${seasonLabel}`}
        description={`${seasonEpisodes.length} ${seasonEpisodes.length === 1 ? 'Episode' : 'Episodes'}`}
        actions={
          <div className="flex items-center gap-2">
            <AdminActions
              mediaType="tv"
              mediaId={showIdNumber}
              mediaTitle={show?.title || ''}
              seasonId={seasonDbId}
              onDeleteNavigate="/tv"
            />
            <Button onClick={handleBackClick}>← Back to Show</Button>
          </div>
        }
      />

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

      <BatchProgressProvider mediaIds={episodeIds}>
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {seasonEpisodes.map((episode: TVEpisodeResponse) => (
            <div key={episode.id} className="space-y-1">
              <EpisodeCard
                episode={episode}
                onClick={() => handlePlayEpisode(episode)}
              />
              <EpisodeAdminRow episodeId={episode.id ?? 0} />
            </div>
          ))}
        </div>
      </BatchProgressProvider>
    </div>
  )
}

// Stricter runtime parser types for TanStack Router ValidateSearch
interface SearchParams {
  episodeId?: number
  t?: number
}

export const Route = createFileRoute('/_layout/tv/$showId/season/$seasonNumber')({
  component: SeasonDetail,
  validateSearch: (search: Record<string, unknown>): SearchParams => {
    const epId = Number(search.episodeId)
    const tPos = Number(search.t)
    return {
      episodeId: !isNaN(epId) ? epId : undefined,
      t: !isNaN(tPos) ? tPos : undefined,
    }
  },
})