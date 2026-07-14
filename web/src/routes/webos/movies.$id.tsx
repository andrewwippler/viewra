import { createFileRoute } from '@tanstack/react-router'
import { TvMovieDetail } from '@/views/movies/TvMovieDetail'

const MovieDetail = () => {
  const { id } = Route.useParams()
  const movieId = parseInt(id, 10)

  return <TvMovieDetail movieId={movieId} />
}

export const Route = createFileRoute('/webos/movies/$id')({
  component: MovieDetail,
  validateSearch: (search: Record<string, unknown>) => {
    const t = search.t
    const parsedT = typeof t === 'string' ? parseInt(t, 10) : typeof t === 'number' ? t : undefined
    return {
      t: parsedT !== undefined && !isNaN(parsedT) ? parsedT : undefined,
    }
  },
})
