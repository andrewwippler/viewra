import { createFileRoute } from '@tanstack/react-router'
import { TvSeasonDetail } from '@/views/tv/TvSeasonDetail'

const SeasonDetail = () => {
  const { showId, seasonNumber } = Route.useParams()
  const search = Route.useSearch() as { episodeId?: number; t?: number }
  const showIdNumber = parseInt(showId, 10)
  const seasonNum = parseInt(seasonNumber, 10)

  return (
    <TvSeasonDetail
      showId={showIdNumber}
      seasonNumber={seasonNum}
      episodeId={search.episodeId}
      initialTime={search.t}
    />
  )
}

export const Route = createFileRoute('/webos/tv/$showId/season/$seasonNumber')({
  component: SeasonDetail,
  validateSearch: (search: Record<string, unknown>) => {
    const episodeId = search.episodeId
    const parsedId = typeof episodeId === 'string' ? parseInt(episodeId, 10) : typeof episodeId === 'number' ? episodeId : undefined
    const t = search.t
    const parsedT = typeof t === 'string' ? parseInt(t, 10) : typeof t === 'number' ? t : undefined
    return {
      episodeId: parsedId && !isNaN(parsedId) ? parsedId : undefined,
      t: parsedT !== undefined && !isNaN(parsedT) ? parsedT : undefined,
    }
  },
})
