import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { VideoPlayerContainer } from '@/components/media'
import { useCallback, useEffect, useState } from 'react'
import { Button, Card, CardContent } from '@/components/ui'
import { PageHeader, LoadingPage, ErrorPage } from '@/components/common'
import { moviesApi } from '@/lib/api/movies'
import { useMediaPlayback, useMovieImages } from '@/lib/hooks'
import { logger } from '@/lib/utils/logger'
import { getPosterImage, getImageUrl } from '@/lib/types/images'
import type { GithubComMantonxViewraInternalApplicationMoviesMovieResponse, GithubComMantonxViewraInternalApplicationMediaMediaResponse } from '@/lib/api/generated/models'
import type { ViewMode } from '@/components/common'

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

  // Fetch movie poster
  const movieImages = useMovieImages(movieId, { enabled: !!movieId })

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
      await playMedia(movie.id, movie as GithubComMantonxViewraInternalApplicationMediaMediaResponse, startTime)
    } catch (err) {
      logger.error('Failed to play movie:', err)
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
        actions={<Button onClick={handleBackClick}>← Back to Movies</Button>}
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
