/**
 * TV Movie Detail Page
 *
 * TV-friendly interface with large play button that's auto-focused.
 * Shows movie poster, title, and big play button.
 *
 * Navigation: Uses linear tab-order navigation (matches jellyfin-webos).
 * Up/Down arrows move between focusable elements in DOM order.
 */

import { useEffect, useRef, useState, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Play, ArrowLeft, Info } from 'lucide-react'
import { moviesApi } from '@/lib/api/movies'
import { useMediaPlayback, useMovieImages, useMediaProgress, useWebOSInputNavigation } from '@/lib/hooks'
import { VideoPlayerContainer, RatingButtons, MarkWatchedButton } from '@/components/media'
import { getPosterImage, getImageUrl } from '@/lib/types/images'
import { logger } from '@/lib/utils/logger'
import { cn } from '@/lib/utils'
import type { GithubComMantonxViewraInternalApplicationMoviesMovieResponse, GithubComMantonxViewraInternalApplicationMediaMediaResponse } from '@/lib/api/generated/models'

interface TvMovieDetailProps {
  movieId: number
  onBack?: () => void
}

export const TvMovieDetail = ({ movieId, onBack }: TvMovieDetailProps) => {
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
        navigate({ to: '/webos' })
      }
    },
  })

  // State for movie data
  const [movie, setMovie] = useState<GithubComMantonxViewraInternalApplicationMoviesMovieResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showDetails, setShowDetails] = useState(false)

  const playButtonRef = useRef<HTMLButtonElement>(null)

  // Fetch movie poster
  const movieImages = useMovieImages(movieId, { enabled: !!movieId })
  const { data: progress } = useMediaProgress(movieId)

  // Calculate progress values
  const progressPosition = progress?.progress_seconds ?? 0
  const progressDuration = progress?.duration_seconds ?? 0
  const hasProgress = progressPosition > 0 && progressDuration > 0

  // Fetch movie data
  const loadMovie = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await moviesApi.getMovie(movieId)
      if (response.status === 200 && response.data) {
        setMovie(response.data)
      } else {
        setError('Failed to load movie details')
      }
    } catch (err) {
      logger.error('Failed to load movie:', err)
      setError('Failed to load movie details')
    } finally {
      setIsLoading(false)
    }
  }, [movieId])

  // Load movie on mount
  useEffect(() => {
    loadMovie()
  }, [loadMovie])

  const handlePlayMovie = async () => {
    if (!movie) {return}
    try {
      await playMedia(movie.id, movie as GithubComMantonxViewraInternalApplicationMediaMediaResponse, progressPosition)
    } catch (err) {
      logger.error('Failed to play movie:', err)
    }
  }

  const handleClosePlayer = () => {
    stopPlayback()
  }

  const handleTimeUpdate = (time: number) => {
    if (time > 0) {
      navigate({
        to: `/webos/movies/${movieId}`,
        search: { t: Math.floor(time) },
        replace: true,
      })
    }
  }

  const handleBack = () => {
    if (onBack) {
      onBack()
    } else {
      window.history.back()
    }
  }

  const handleBackClick = () => {
    navigate({ to: '/webos' })
  }

  // Show video player when playing
  if (playbackState.isPlaying && movie) {
    return (
      <VideoPlayerContainer
        playbackState={playbackState}
        media={movie}
        onClose={handleClosePlayer}
        onTimeUpdate={handleTimeUpdate}
        onQualityChange={changeQuality}
      />
    )
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-neutral-900 flex items-center justify-center">
        <div className="animate-pulse text-neutral-400 text-2xl">Loading...</div>
      </div>
    )
  }

  if (error || !movie) {
    return (
      <div className="min-h-screen bg-neutral-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-error-400 text-2xl mb-4">{error || 'Movie not found'}</p>
          <button onClick={handleBack} className="px-6 py-3 bg-neutral-800 rounded-lg text-white">
            Go Back
          </button>
        </div>
      </div>
    )
  }

  // Get poster URL
  const images = movieImages.data?.images || []
  const poster = getPosterImage(images)
  const posterUrl = poster ? getImageUrl(poster.id, 'large') : null

  const runtime = movie.runtime_minutes ? `${Math.floor(movie.runtime_minutes / 60)}h ${movie.runtime_minutes % 60}m` : null
  const year = movie.year ? String(movie.year) : null
  const genres = movie.genre?.join(' • ') || null

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-900 via-neutral-900 to-neutral-950 text-white relative overflow-y-auto">
      {/* Header row */}
      <div data-row className="p-4 flex items-center gap-4 sticky top-0 bg-neutral-900/95 z-10">
        <button
          onClick={handleBackClick}
          className="p-3 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50 transition-colors"
        >
          <ArrowLeft className="w-7 h-7" />
        </button>
        <h1 className="text-2xl font-bold truncate">{movie.title}</h1>
      </div>

      {/* Content row */}
      <div data-row className="px-6 pb-24">
        {/* Poster and info */}
        <div className="flex flex-col lg:flex-row gap-6 items-center lg:items-start">
          {/* Poster */}
          <div className="shrink-0">
            {posterUrl ? (
              <img
                src={posterUrl}
                alt={movie.title}
                className="w-72 h-108 object-cover rounded-xl shadow-2xl"
              />
            ) : (
              <div className="w-72 h-108 bg-neutral-800 rounded-xl flex items-center justify-center text-6xl text-neutral-600">
                🎬
              </div>
            )}
          </div>

          {/* Info panel */}
          <div className="flex-1 text-center lg:text-left">
            <h2 className="text-4xl font-bold mb-2">{movie.title}</h2>

            {/* Metadata row */}
            <div className="flex flex-wrap justify-center lg:justify-start gap-2 text-neutral-400 mb-4">
              {year && <span>{year}</span>}
              {runtime && <span>• {runtime}</span>}
              {genres && <span>• {genres}</span>}
            </div>

            {/* Progress indicator */}
            {hasProgress && (
              <div className="mb-4">
                <div className="w-48 h-2 bg-neutral-700 rounded-full mx-auto lg:mx-0">
                  <div
                    className="h-full bg-primary-500 rounded-full"
                    style={{ width: `${(progressPosition / progressDuration) * 100}%` }}
                  />
                </div>
                <p className="text-base text-neutral-400 mt-1">
                  {Math.floor(progressPosition / 60)}m watched
                </p>
              </div>
            )}

            {/* Plot */}
            {movie.plot && (
              <p className="text-neutral-300 text-base max-w-xl mb-6 line-clamp-3">
                {movie.plot}
              </p>
            )}

            {/* Action buttons */}
            <div className="flex flex-wrap gap-4 justify-center lg:justify-start">
              <button
                ref={playButtonRef}
                id="connect"
                onClick={handlePlayMovie}
                className={cn(
                  'flex items-center gap-3 px-8 py-4 rounded-xl text-2xl font-bold',
                  'bg-primary-600 hover:bg-primary-500 transition-all',
                  'focus:ring-4 focus:ring-primary-500/50'
                )}
              >
                <Play className="w-10 h-10" />
                {progressPosition > 60 ? 'Resume' : 'Play'}
              </button>

              {hasProgress && (
                <button
                  onClick={async () => {
                    if (!movie) {return}
                    try {
                      await playMedia(movie.id, movie as GithubComMantonxViewraInternalApplicationMediaMediaResponse, 0)
                    } catch (err) {
                      logger.error('Failed to restart movie:', err)
                    }
                  }}
                  className={cn(
                    'flex items-center gap-2 px-6 py-4 rounded-xl text-xl font-bold',
                    'bg-neutral-700/50 hover:bg-neutral-600/50 transition-colors',
                    'focus:ring-4 focus:ring-primary-500/50'
                  )}
                >
                  Restart
                </button>
              )}

              <button
                onClick={() => setShowDetails(!showDetails)}
                className={cn(
                  'flex items-center gap-2 px-6 py-4 rounded-xl',
                  'bg-neutral-700/50 hover:bg-neutral-600/50 transition-colors'
                )}
              >
                <Info className="w-6 h-6" />
                Details
              </button>

              <div className="flex items-center gap-2">
                <MarkWatchedButton
                  mediaId={movieId}
                  isWatched={progress ? progress.is_watched || false : false}
                  variant="icon"
                />
                <RatingButtons
                  entityType="movie"
                  entityId={movieId}
                  size="lg"
                  hideOnTV={false}
                />
              </div>
            </div>
          </div>
        </div>

        {/* Extended details (collapsible) */}
        {showDetails && (
          <div className="mt-8 space-y-4">
            {movie.plot && (
              <div className="bg-neutral-800/30 rounded-xl p-4">
                <h3 className="font-semibold mb-2">Plot</h3>
                <p className="text-neutral-300 text-base">{movie.plot}</p>
              </div>
            )}

            {(movie.director || (movie.cast && movie.cast.length > 0)) && (
              <div className="bg-neutral-800/30 rounded-xl p-4">
                <h3 className="font-semibold mb-2">Cast & Crew</h3>
                {movie.director && <p className="text-neutral-300 text-base mb-2"><span className="text-neutral-400">Director:</span> {movie.director}</p>}
                {movie.cast && movie.cast.length > 0 && <p className="text-neutral-300 text-base">{movie.cast.slice(0, 5).join(', ')}</p>}
              </div>
            )}

            {movie.content_rating && (
              <div className="bg-neutral-800/30 rounded-xl p-4">
                <h3 className="font-semibold mb-2">Details</h3>
                <p className="text-neutral-300 text-base">Content Rating: {movie.content_rating}</p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer hint */}
      <div className="fixed bottom-0 left-0 right-0 p-4 text-center text-neutral-500 text-base bg-neutral-900/80">
        Use arrow keys to navigate • OK to select • Back to go back
      </div>
    </div>
  )
}