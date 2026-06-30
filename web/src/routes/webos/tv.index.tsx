import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { Tv, ArrowLeft } from 'lucide-react'
import { useInfiniteTVShows, flattenTVShows } from '@/lib/hooks/useInfiniteTVShows'
import { useLibraryFilter, useWebOSInputNavigation } from '@/lib/hooks'
import { MediaPoster } from '@/components/media/MediaPoster'
import type { GithubComMantonxViewraInternalApplicationTvTVShowSummary } from '@/lib/api/generated/models'
import { breakTitle } from '@/lib/utils'

const chunk = <T,>(arr: T[], size: number): T[][] =>
  Array.from({ length: Math.ceil(arr.length / size) }, (_, i) =>
    arr.slice(i * size, i * size + size)
  )

const WebosTvShows = () => {
  const navigate = useNavigate()

  useWebOSInputNavigation({
    enabled: true,
    rowSelector: '[data-row]',
    onBackPress: () => navigate({ to: '/webos' }),
  })

  const { libraryId } = useLibraryFilter('tv')

  const { data, isLoading, error } = useInfiniteTVShows({ libraryId, sort: 'title_asc', pageSize: 100 })
  const shows = data ? flattenTVShows(data.pages as Array<{ shows?: GithubComMantonxViewraInternalApplicationTvTVShowSummary[] }>) : []

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-900 to-neutral-950 text-white overflow-y-auto">
      {/* Header row */}
      <div data-row className="p-4 flex items-center gap-4 sticky top-0 bg-neutral-900/95 z-10">
        <button
          onClick={() => navigate({ to: '/webos' })}
          className="p-3 bg-neutral-800/50 rounded-xl hover:bg-neutral-700/50 transition-colors"
        >
          <ArrowLeft className="w-6 h-6" />
        </button>
        <h1 className="text-2xl font-bold">TV Shows</h1>
      </div>

      <div className="px-1 pb-6">
        {isLoading ? (
          <div data-row className="grid grid-cols-5 gap-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i}>
                <div className="w-full min-h-[360px] bg-neutral-800/60 rounded-lg skeleton" />
                <div className="mt-1 h-4 w-24 bg-neutral-800/60 rounded skeleton" />
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="text-center text-neutral-400 py-20">
            <p>Failed to load TV shows</p>
          </div>
        ) : shows.length === 0 ? (
          <div className="text-center text-neutral-500 py-20">
            <Tv className="w-8 h-8 mx-auto mb-4 opacity-50" />
            <p className="text-xl mb-2">No TV shows found</p>
            <p className="text-base">Try a different search or add TV shows to your library</p>
          </div>
        ) : (
          chunk(shows, 5).map((group, rowIndex) => (
            <div data-row key={rowIndex} className="grid grid-cols-5 gap-2">
              {group.map((show: GithubComMantonxViewraInternalApplicationTvTVShowSummary) => (
                <button
                  key={show.id}
                  type="button"
                  id={rowIndex === 0 && group.indexOf(show) === 0 ? 'connect' : undefined}
                  className="border-none bg-transparent text-left cursor-pointer p-0.5"
                  onClick={() => show.id && navigate({ to: `/webos/tv/${show.id}` })}
                >
                  <div className="relative rounded-lg overflow-hidden bg-neutral-800 w-full min-h-[360px]">
                    <MediaPoster
                      mediaId={show.id ?? 0}
                      mediaType="tv-show"
                      alt={show.title || 'Untitled'}
                      preset="medium"
                      fallbackIcon="📺"
                      className="h-full w-full object-cover"
                    />
                  </div>
                  <div className="mt-1 px-0.5 min-h-[2rem]">
                    <h3 className="text-white text-sm font-medium w-full truncate" dangerouslySetInnerHTML={{ __html: breakTitle(show.title || 'Untitled') }} />
                  </div>
                </button>
              ))}
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export const Route = createFileRoute('/webos/tv/')({
  component: WebosTvShows,
})
