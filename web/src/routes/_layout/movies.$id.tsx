import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { VideoPlayerContainer } from '@/components/media'
import { useCallback, useEffect, useState } from 'react'
import { Button, Card, CardContent } from '@/components/ui'
import { PageHeader, LoadingPage, ErrorPage } from '@/components/common'
import { moviesApi } from '@/lib/api/movies'
import { useMediaPlayback, useMovieImages, useMarkWatched, useMarkUnwatched } from '@/lib/hooks'
import { logger } from '@/lib/utils/logger'
import { getPosterImage, getImageUrl } from '@/lib/types/images'
import type { GithubComMantonxViewraInternalApplicationMoviesMovieResponse, GithubComMantonxViewraInternalApplicationMediaMediaResponse, GithubComMantonxViewraInternalApplicationMoviesMediaVariantResponse } from '@/lib/api/generated/models'
import type { ViewMode } from '@/components/common'
import { useMediaProgress } from '@/lib/hooks'
import { AdminActions } from '@/features/nitpicky-edits'

const MovieDetail = () => {
  const navigate = useNavigate()
  const { playMedia, stopPlayback, changeQuality, playbackState } = useMediaPlayback()
  const { id } = Route.useParams()
  const search = Route.useSearch()
  const movieId = parseInt(id, 10)

  // State for movie data
  const [movie, setMovie] = useState<GithubComMantonxViewraInternalApplicationMoviesMovieResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeVariantId, setActiveVariantId] = useState<number | null>(null)

  // Fetch movie poster
  const movieImages = useMovieImages(movieId, { enabled: !!movieId })
  const { data: progress } = useMediaProgress(movieId)

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

  // Load movie on mount or when ID changes
  useEffect(() => {
    loadMovie()
  }, [loadMovie])

  const handleBackClick = () => {
    navigate({ to: '/movies', search: { id: undefined, t: undefined, q: undefined, sort: undefined, genres: undefined, yearMin: undefined, yearMax: undefined, qualities: undefined, watched: undefined, view: undefined } })
  }

  const handlePlayMovie = async (startTime?: number) => {
    if (!movie) {return}
    try {
      const targetId = activeVariantId || movie.id
      const mediaPayload: GithubComMantonxViewraInternalApplicationMediaMediaResponse = {
        id: targetId,
        title: movie.title,
        file_path: movie.file_path,
        file_size: movie.file_size,
        duration: movie.duration,
        width: movie.width,
        height: movie.height,
        video_codec: movie.video_codec,
        audio_codec: movie.audio_codec,
        container_format: movie.container_format,
        bitrate: movie.bitrate,
        frame_rate: movie.frame_rate,
        library_id: movie.library_id,
        is_extra: movie.is_extra,
        created_at: movie.created_at,
        updated_at: movie.updated_at,
      }
      await playMedia(targetId, mediaPayload, startTime)
    } catch (err) {
      logger.error('Failed to play movie:', err)
    }
  }

  const markWatched = useMarkWatched()
  const markUnwatched = useMarkUnwatched()

  const handleToggleWatched = () => {
    if (progress?.is_watched) {
      markUnwatched.mutate({ media_id: movieId })
    } else {
      markWatched.mutate({ media_id: movieId })
    }
  }

  // Determine if we should show the video player (when URL has playback params)
  const shouldShowPlayer = search.t !== undefined

  const handleClosePlayer = () => {
    stopPlayback()
    navigate({ to: `/movies/${movieId}`, search: { t: undefined, q: undefined, sort: undefined, genres: undefined, yearMin: undefined, yearMax: undefined, qualities: undefined, watched: undefined, view: undefined } })
  }

  const videoPlayer = (
    <div className="p-8">
      <PageHeader
        title={movie?.title || 'Loading...'}
        actions={
          <div className="flex items-center gap-2">
            <Button onClick={handleClosePlayer} variant="secondary" size="sm">← Back to Details</Button>
          </div>
        }
      />
      <VideoPlayerContainer
        playbackState={playbackState}
        media={movie}
        onClose={handleClosePlayer}
        onTimeUpdate={(time: number) => {
          navigate({
            to: `/movies/${movieId}`,
            search: { t: Math.floor(time) },
            replace: true
          })
        }}
        onQualityChange={changeQuality}
      />
    </div>
  )

  if (isLoading) {
    return <LoadingPage text="Loading movie details..." />
  }

  if (error) {
    return <ErrorPage error={error} context="movie details" />
  }

  if (!movie) {
    return <ErrorPage error="Movie not found" context="movie details" />
  }

  // If we have playback parameters in URL, show the player
  if (shouldShowPlayer) {
    return videoPlayer
  }

  const movieTitle = movie.title || 'Unknown Title'
  const movieYear = movie.year ? ` (${movie.year})` : ''
  const subtitleParts: string[] = []
  if (movie.year) {subtitleParts.push(String(movie.year))}
  if (movie.original_language) {subtitleParts.push(movie.original_language.toUpperCase())}
  if (movie.genre && movie.genre.length > 0) {subtitleParts.push(movie.genre.join(', '))}
  const subtitle = subtitleParts.length > 0 ? subtitleParts.join(' • ') : ''

  return (
    <div className="p-8">
      <PageHeader
        title={`${movieTitle}${movieYear}`}
        description={subtitle}
        actions={
          <div className="flex items-center gap-2">
            <div className="relative">
              <AdminActions
                mediaType="movie"
                mediaId={movieId}
                mediaTitle={movie?.title || ''}
                isWatched={progress?.is_watched}
                onDeleteNavigate="/movies"
              />
            </div>
            <button
              onClick={handleToggleWatched}
              className={`p-1.5 rounded-lg transition-colors cursor-pointer hover:bg-neutral-100 dark:hover:bg-white/10 ${
                progress?.is_watched
                  ? 'text-green-500 hover:text-green-600 dark:text-green-400 dark:hover:text-green-300'
                  : 'text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'
              }`}
              aria-label={progress?.is_watched ? 'Mark as unwatched' : 'Mark as watched'}
              title={progress?.is_watched ? 'Mark as unwatched' : 'Mark as watched'}
            >
              {progress?.is_watched ? (
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="lucide lucide-check-circle">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
              ) : (
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="lucide lucide-check-circle">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
              )}
            </button>
            <Button onClick={handleBackClick}>← Back to Movies</Button>
          </div>
        }
      />
      
      {/* Movie metadata - two column layout: poster left, details right */}
      <div className="grid grid-cols-1 lg:grid-cols-[300px_1fr] gap-8">
        {/* Left column: Poster */}
        <div className="flex flex-col items-center lg:items-start">
          <Card className="w-full max-w-[300px]">
            <CardContent className="flex flex-col items-center">
              {/* Movie poster */}
              {(() => {
                const images = movieImages.data?.images || []
                const poster = getPosterImage(images)
                const posterUrl = poster ? getImageUrl(poster.id, 'medium') : null
                return posterUrl ? (
                  <img
                    src={posterUrl}
                    alt={`${movie.title} poster`}
                    className="w-full max-w-[250px] h-auto object-cover rounded-lg shadow-lg"
                  />
                ) : (
                  <div className="w-full max-w-[250px] aspect-[2/3] bg-neutral-200 dark:bg-neutral-700 flex items-center justify-center text-neutral-500 dark:text-neutral-400 rounded-lg">
                    🎬
                  </div>
                )
              })()}
              <h1 className="text-2xl font-bold mt-4 text-center">{movie.title}</h1>
              {movie.year && <p className="text-muted-foreground">{movie.year}</p>}
              {movie.original_language && <p className="text-muted-foreground">{movie.original_language.toUpperCase()}</p>}
              {movie.genre && movie.genre.length > 0 && (
                <p className="text-muted-foreground text-center">{movie.genre.join(', ')}</p>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Right column: Details */}
        <div className="space-y-6">
          {/* Plot */}
          {movie.plot && (
            <Card>
              <CardContent>
                <h2 className="font-semibold mb-2">Plot</h2>
                <p className="text-muted-foreground">{movie.plot}</p>
              </CardContent>
            </Card>
          )}

          {/* Language variants */}
          {movie.variants && movie.variants.length > 0 && (
            <Card>
              <CardContent>
                <h2 className="font-semibold mb-2">Language</h2>
                <select
                  value={activeVariantId || ''}
                  onChange={(e) => setActiveVariantId(e.target.value ? Number(e.target.value) : null)}
                  className="w-full px-3 py-2 rounded-lg border border-neutral-300 dark:border-neutral-600 bg-white dark:bg-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">Original</option>
                  {movie.variants.map((v: GithubComMantonxViewraInternalApplicationMoviesMediaVariantResponse) => (
                    <option key={v.id} value={v.id}>
                      {v.language ? v.language.toUpperCase() : 'Unknown'}
                    </option>
                  ))}
                </select>
              </CardContent>
            </Card>
          )}

          {/* Cast and Crew */}
          {(movie.director || movie.cast && movie.cast.length > 0) && (
            <Card>
              <CardContent>
                <h2 className="font-semibold mb-2">Cast & Crew</h2>
                {movie.director && (
                  <div className="mb-2">
                    <span className="font-medium">Director:</span> {movie.director}
                  </div>
                )}
                {movie.cast && movie.cast.length > 0 && (
                  <>
                    <span className="font-medium mb-1 block">Cast:</span>
                    <p className="text-muted-foreground">{movie.cast.slice(0, 5).join(', ')}{movie.cast.length > 5 && '...'}</p>
                  </>
                )}
              </CardContent>
            </Card>
          )}

          {/* Technical specs */}
          {(movie.runtime_minutes || movie.content_rating) && (
            <Card>
              <CardContent>
                <h2 className="font-semibold mb-2">Details</h2>
                <div className="space-y-2">
                  {movie.runtime_minutes && (
                    <div className="flex justify-between">
                      <span>Runtime:</span>
                      <span className="text-muted-foreground">{movie.runtime_minutes} min</span>
                    </div>
                  )}
                  {movie.content_rating && (
                    <div className="flex justify-between">
                      <span>Content Rating:</span>
                      <span className="text-muted-foreground">{movie.content_rating}</span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Action buttons */}
      <div className="mt-6">
        <div className="flex flex-wrap gap-4 justify-center">
          <Button
            onClick={() => { handlePlayMovie(0) }}
            variant="primary"
          >
            Play
          </Button>
        </div>
      </div>

    </div>
  )
}

export const Route = createFileRoute('/_layout/movies/$id')({
  component: MovieDetail,
  validateSearch: (search: Record<string, unknown>) => {
    const t = search.t
    const parsedT = typeof t === 'string' ? parseInt(t, 10) : typeof t === 'number' ? t : undefined
    const q = typeof search.q === 'string' ? search.q : undefined
    const sort = typeof search.sort === 'string' ? search.sort : undefined
    const genres = typeof search.genres === 'string' ? search.genres : undefined
    const yearMin = typeof search.yearMin === 'number' ? search.yearMin : typeof search.yearMin === 'string' ? parseInt(search.yearMin, 10) : undefined
    const yearMax = typeof search.yearMax === 'number' ? search.yearMax : typeof search.yearMax === 'string' ? parseInt(search.yearMax, 10) : undefined
    const qualities = typeof search.qualities === 'string' ? search.qualities : undefined
    const watched = typeof search.watched === 'string' ? search.watched : undefined
    const view = typeof search.view === 'string' && (search.view === 'grid' || search.view === 'list') ? search.view as ViewMode : undefined

    return {
      t: parsedT && !isNaN(parsedT) ? parsedT : undefined,
      q,
      sort,
      genres,
      yearMin: yearMin && !isNaN(yearMin) ? yearMin : undefined,
      yearMax: yearMax && !isNaN(yearMax) ? yearMax : undefined,
      qualities,
      watched,
      view,
    }
  },
})
