/**
 * TV Show Detail Page
 * 
 * TV-friendly interface showing seasons as clickable cards.
 * Navigation: Grid navigation with up/down moving between rows, left/right moving between columns.
 */

import { useRef } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Play } from 'lucide-react'
import { tvApi } from '@/lib/api/tv'
import { getPosterImage, getFanartImage, getImageUrl } from '@/lib/types/images'
import { useWebOSInputNavigation } from '@/lib/hooks'
import { cn } from '@/lib/utils'
import { useTVShowImages } from '@/lib/hooks'
import type { SeasonGroup } from '@/lib/types/tv'
import type { GithubComMantonxViewraInternalApplicationTvTVEpisodeResponse } from '@/lib/api/generated/models'

const chunk = <T,>(arr: T[], size: number): T[][] =>
  Array.from({ length: Math.ceil(arr.length / size) }, (_, i) =>
    arr.slice(i * size, i * size + size)
  )

interface TvShowDetailProps {
  showId: number
  onBack?: () => void
}

export const TvShowDetail = ({ showId, onBack }: TvShowDetailProps) => {
  const navigate = useNavigate()
  const playNextRef = useRef<HTMLButtonElement>(null)

  useWebOSInputNavigation({
    enabled: true,
    rowSelector: '[data-row]',
    onBackPress: () => {
      if (onBack) {
        onBack()
      } else {
        navigate({ to: '/webos' })
      }
    },
  })

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

  const {
    data: imagesData,
    isLoading: isLoadingImages,
  } = useTVShowImages(showId)

  const show = showData?.data && 'title' in showData.data ? showData.data : null
  const isLoading = isLoadingShow || isLoadingEpisodes || isLoadingImages
  const error = showError || episodesError

  // Group episodes by season
  const seasons: SeasonGroup[] = []
  if (episodesData?.data && 'episodes' in episodesData.data) {
    const allEpisodes = episodesData.data.episodes || []
    const seasonMap = new Map<number, SeasonGroup>()
    
    allEpisodes.forEach((episode: GithubComMantonxViewraInternalApplicationTvTVEpisodeResponse) => {
      const seasonNum = episode.season ?? 0
      if (!seasonMap.has(seasonNum)) {
        seasonMap.set(seasonNum, {
          season: seasonNum,
          season_id: episode.season_id,
          episode_count: 0,
          episodes: [],
        })
      }
      const seasonGroup = seasonMap.get(seasonNum)
      if (seasonGroup && seasonGroup.episode_count !== undefined) {
        seasonGroup.episodes.push(episode)
        seasonGroup.episode_count++
      }
    })
    
    // Sort seasons (0 at the end, rest in ascending order)
    seasons.push(...Array.from(seasonMap.values()).sort((a, b) => {
      const aSeason = a.season ?? 0
      const bSeason = b.season ?? 0
      if (aSeason === 0) {return 1}
      if (bSeason === 0) {return -1}
      return aSeason - bSeason
    }))
  }

  const handleSeasonClick = (seasonNumber: number) => {
    navigate({ to: `/webos/tv/${showId}/season/${seasonNumber}` })
  }

  const handleBackClick = () => {
    if (onBack) {
      onBack()
    } else {
      navigate({ to: '/webos' })
    }
  }

  const handlePlayNextEpisode = async () => {
    try {
      const nextEpisode = await tvApi.getNextEpisode(showId)
      navigate({
        to: `/webos/tv/${showId}/season/${nextEpisode.season}`,
        search: { episodeId: nextEpisode.id }
      })
    } catch (err) {
      console.error('Failed to get next episode:', err)
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

  // Get images
  const showImages = imagesData?.images || []
  const backdrop = getFanartImage(showImages)
  const backdropUrl = backdrop ? getImageUrl(backdrop.id, 'large') : null
  const poster = getPosterImage(showImages)
  const posterUrl = poster ? getImageUrl(poster.id, 'large') : null

  const totalEpisodes = seasons.reduce((sum, s) => sum + (s.episode_count || 0), 0)

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-900 via-neutral-900 to-neutral-950 text-white relative overflow-y-auto">
      {/* Subtle backdrop behind gradient */}
      {backdropUrl && (
        <div className="fixed inset-0 z-[-1]">
          <img src={backdropUrl} alt="" className="w-full h-full object-cover opacity-30" />
        </div>
      )}

      {/* Content */}
      <div className="relative z-10 pb-24">
        {/* Header row */}
        <div data-row className="p-4 flex items-center gap-4 sticky top-0 bg-neutral-900/95 z-10">
          <button
            onClick={handleBackClick}
            className="p-3 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50 transition-colors"
          >
            <ArrowLeft className="w-7 h-7" />
          </button>
          <h1 className="text-2xl font-bold truncate">{show.title}</h1>
        </div>

        {/* Info row */}
        <div data-row className="px-6 pb-6">
          <div className="flex items-center gap-4 mb-4">
            {posterUrl && (
              <img src={posterUrl} alt={show.title} className="w-28 h-42 object-cover rounded-lg" />
            )}
            <div>
              <h2 className="text-3xl font-bold">{show.title}</h2>
              <p className="text-neutral-400">
                {seasons.length} {seasons.length === 1 ? 'Season' : 'Seasons'} • {totalEpisodes} Episodes
              </p>
              {show.genre && show.genre.length > 0 && (
                <p className="text-neutral-400 text-base mt-1">{show.genre.join(' • ')}</p>
              )}
            </div>
          </div>

          {/* Quick play button */}
          <button
            ref={playNextRef}
            onClick={handlePlayNextEpisode}
            className={cn(
              'flex items-center gap-2 px-6 py-3 rounded-xl mb-6',
              'bg-primary-600 hover:bg-primary-500 transition-colors'
            )}
          >
            <Play className="w-6 h-6" />
            Play Next Episode
          </button>

          {/* Plot */}
          {show.plot && (
            <p className="text-neutral-300 text-base mb-6 line-clamp-2">{show.plot}</p>
          )}
        </div>

        {/* Seasons */}
        <div className="px-4 pb-6">
          <h3 className="text-xl font-semibold mb-4">Seasons</h3>
          {chunk(seasons, 5).map((group, rowIndex) => (
            <div data-row key={rowIndex} className="grid grid-cols-5 gap-2 mb-2">
              {group.map((season) => {
                const seasonLabel = season.season === 0 ? 'Specials' : `Season ${season.season}`
                const seasonPosterUrl = posterUrl

                return (
                  <button
                    key={season.season ?? rowIndex}
                    id={rowIndex === 0 && group.indexOf(season) === 0 ? 'connect' : undefined}
                    onClick={() => handleSeasonClick(season.season ?? 0)}
                    className={cn(
                      'text-left rounded-lg overflow-hidden transition-all bg-neutral-800 relative',
                      'focus:ring-4 focus:ring-primary-500/50'
                    )}
                  >
                    <div className="w-full aspect-[2/3]">
                      {seasonPosterUrl ? (
                        <img src={seasonPosterUrl} alt={seasonLabel} className="w-full h-full object-cover" />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center text-4xl bg-neutral-800">
                          📺
                        </div>
                      )}
                    </div>
                    <div className="p-2">
                      <p className="font-bold text-white text-base truncate">{seasonLabel}</p>
                      <p className="text-neutral-500 text-sm">{season.episode_count} episodes</p>
                    </div>
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Footer hint */}
      <div className="fixed bottom-0 left-0 right-0 p-4 text-center text-neutral-500 text-base bg-neutral-900/80">
        Use arrow keys to navigate • OK to select season • Back to go back
      </div>
    </div>
  )
}