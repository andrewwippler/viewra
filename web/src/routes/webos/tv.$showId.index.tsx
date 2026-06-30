import { createFileRoute } from '@tanstack/react-router'
import { TvShowDetail } from '@/views/tv/TvShowDetail'

const ShowDetail = () => {
  const { showId } = Route.useParams()
  const showIdNumber = parseInt(showId, 10)

  return <TvShowDetail showId={showIdNumber} />
}

export const Route = createFileRoute('/webos/tv/$showId/')({
  component: ShowDetail,
})
