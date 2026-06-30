/**
 * TV Home Page - WebOS Dashboard
 *
 * A proper grid-based dashboard for TV navigation.
 * Navigation: Grid navigation with up/down moving between rows, left/right moving between columns.
 */

import { useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Film, Tv, Music, Radio, ChevronRight } from 'lucide-react'
import { useWebOSInputNavigation } from '@/lib/hooks'
import { useGetApiLibraries } from '@/lib/api/generated/libraries/libraries'
import { useHomeSections } from '@/lib/hooks/useWidgets'
import {
  WidgetType,
  type ContinueWatchingData,
  type ContinueWatchingItem,
  type MediaRowData,
} from '@/components/home/widgets/widget.types'
import { MediaPoster } from '@/components/media/MediaPoster'
import { breakTitle } from '@/lib/utils'

type LibraryTypeString = 'movies' | 'tv' | 'music' | 'live_tv'


const entityTypeToMediaType = (type: string) => (type === 'tv_show' ? 'tv-show' : type)

interface GridCardProps {
  id?: string
  mediaId: number
  mediaType: 'movie' | 'tv-show'
  title: string
  subtitle?: string
  progress?: number
  onClick: () => void
}

const GridCard = ({ id, mediaId, mediaType, title, subtitle, progress, onClick }: GridCardProps) => {
  return (
    <button
      id={id}
      type="button"
      className="border-none bg-transparent text-left cursor-pointer p-0.5"
      onClick={onClick}
    >
      <div className="relative rounded-lg overflow-hidden bg-neutral-800 w-full aspect-[2/3]">
        <MediaPoster
          mediaId={mediaId}
          mediaType={mediaType}
          alt={title}
          preset="medium"
          fallbackIcon={mediaType === 'movie' ? '🎬' : '📺'}
          className="h-full w-full object-cover"
        />
        {progress !== undefined && progress > 0 && (
          <div className="absolute bottom-0 left-0 right-0 h-1 bg-neutral-700/80">
            <div className="h-full bg-primary-500" style={{ width: `${progress}%` }} />
          </div>
        )}
      </div>
      <div className="mt-1 px-0.5 min-h-[2rem]">
        <h3 className="text-white text-xs font-medium w-full truncate" dangerouslySetInnerHTML={{ __html: breakTitle(title) }} />
        {subtitle && <p className="text-neutral-500 text-xs truncate">{subtitle}</p>}
      </div>
    </button>
  )
}

interface SectionHeaderProps {
  title: string
  onSeeAll?: () => void
}

const SectionHeader = ({ title, onSeeAll }: SectionHeaderProps) => {
  return (
    <div className="flex items-center justify-between mb-3 px-1">
      <h2 className="text-white text-sm font-semibold">{title}</h2>
      {onSeeAll && (
        <button
          type="button"
          onClick={onSeeAll}
          className="flex items-center text-neutral-400 hover:text-white text-xs transition-colors"
        >
          See All <ChevronRight className="w-4 h-4 ml-1" />
        </button>
      )}
    </div>
  )
}

interface CategoryButtonProps {
  id?: string
  icon: React.ReactNode
  label: string
  onClick: () => void
}

const CategoryButton = ({ id, icon, label, onClick }: CategoryButtonProps) => {
  return (
    <button
      id={id}
      type="button"
      className="border-none bg-neutral-800/60 hover:bg-neutral-700/60 text-left cursor-pointer p-6 rounded-xl transition-colors min-w-[220px]"
      onClick={onClick}
    >
      <div className="flex flex-col items-center justify-center gap-3">
        <span className="text-primary-400 w-20 h-20 flex items-center justify-center">{icon}</span>
        <span className="font-semibold text-white text-lg text-center">{label}</span>
      </div>
    </button>
  )
}

const TvHomeSkeleton = () => (
  <div className="min-h-screen bg-gradient-to-b from-neutral-900 to-neutral-950 p-4">
    <div className="flex gap-4 mb-8">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="w-[140px] h-[80px] bg-neutral-800/60 rounded-lg skeleton" />
      ))}
    </div>
    <div className="grid grid-cols-5 gap-2">
      {Array.from({ length: 10 }).map((_, i) => (
        <div key={i}>
          <div className="w-full aspect-[2/3] bg-neutral-800/60 rounded-lg skeleton" />
          <div className="mt-1 h-3 w-20 bg-neutral-800/60 rounded skeleton" />
        </div>
      ))}
    </div>
  </div>
)

export const TvHome = () => {
  const navigate = useNavigate()

  useWebOSInputNavigation({
    enabled: true,
    rowSelector: '[data-row]',
    onBackPress: () => {
      // At root, back does nothing
    },
  })

  const { data: librariesData } = useGetApiLibraries()
  const { data: homeSections, isLoading } = useHomeSections('tv')

  const libraryInfo: { movies: boolean; tv: boolean; music: boolean; live_tv: boolean } = {
    movies: false,
    tv: false,
    music: false,
    live_tv: false,
  }

	// LGTM - Library type check confirmed
	const libraryList = librariesData?.data && 'libraries' in librariesData.data
    ? librariesData.data.libraries
    : undefined
  if (libraryList && Array.isArray(libraryList)) {
    for (const lib of libraryList) {
      const libType = lib.type as LibraryTypeString
      if (libType in libraryInfo) {
        libraryInfo[libType] = true
      }
    }
  }

  const sections = homeSections?.sections ?? []

  // Get continue watching items
  const continueSection = sections.find((s) => s.type === WidgetType.ContinueRow)
  const continueData = continueSection?.data as ContinueWatchingData | undefined
  const continueItems: ContinueWatchingItem[] = continueData?.items ?? []

  // Get up next items (from up-next plugin)
  const upNextSection = sections.find((s) => s.id === 'up-next')
  const upNextData = upNextSection?.data as ContinueWatchingData | undefined
  const upNextItems: ContinueWatchingItem[] = upNextData?.items ?? []

  // Get media rows for recommendations
  const mediaSections = sections.filter((s) => s.type === WidgetType.MediaRow)
  const findSection = (keywords: string[]) =>
    mediaSections.find((s) => {
      const title = ((s.data as MediaRowData).title ?? '').toLowerCase()
      return keywords.some((kw) => title.includes(kw))
    })
  const recommendedSection = findSection(['recommend'])
  const newAdditionsSection = findSection(['recently', 'new', 'recent', 'latest', 'recently added', 'new additions']) || mediaSections[1]
  const unwatchedSection = findSection(['unwatched'])

  const handleEntityNavigate = useCallback(
    (entityType: string, entityId: number) => {
      if (entityType === 'movie') {
        navigate({ to: `/webos/movies/${entityId}` } as never)
      } else {
        navigate({ to: `/webos/tv/${entityId}` } as never)
      }
    },
    [navigate]
  )

  const handleContinueItemClick = useCallback(
    (item: ContinueWatchingItem) => {
      if (item.entity_type === 'movie') {
        navigate({ to: `/webos/movies/${item.entity_id}` } as never)
      } else if (item.episode_context) {
        navigate({
          to: `/webos/tv/${item.entity_id}/season/${item.episode_context.season}`,
          search: {
            episodeId: item.episode_context.episode_media_id,
            t: Math.floor(item.progress?.position_seconds ?? 0),
          },
        } as never)
      } else {
        navigate({ to: `/webos/tv/${item.entity_id}` } as never)
      }
    },
    [navigate]
  )

  if (isLoading) {
    return <TvHomeSkeleton />
  }

  const hasLibraries = libraryInfo.movies || libraryInfo.tv || libraryInfo.music || libraryInfo.live_tv

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-900 to-neutral-950 text-white overflow-y-auto">
      {/* Header row */}
      <div data-row className="p-4 sticky top-0 bg-neutral-900/95 z-10">
        <h1 className="text-xl font-bold">ViewRA</h1>
      </div>

      <div className="px-4 pb-6">
        {/* Category buttons row */}
        {hasLibraries && (
          <div data-row className="flex gap-4 mb-8">
            {libraryInfo.movies && (
              <CategoryButton
                id="connect"
                icon={<Film className="w-20 h-20" />}
                label="Movies"
                onClick={() => navigate({ to: '/webos/movies' } as never)}
              />
            )}
            {libraryInfo.tv && (
              <CategoryButton
                icon={<Tv className="w-20 h-20" />}
                label="TV Shows"
                onClick={() => navigate({ to: '/webos/tv' } as never)}
              />
            )}
            {libraryInfo.music && (
              <CategoryButton
                icon={<Music className="w-20 h-20" />}
                label="Music"
                onClick={() => {}} // Music not yet implemented
              />
            )}
            {libraryInfo.live_tv && (
              <CategoryButton
                icon={<Radio className="w-20 h-20" />}
                label="Live TV"
                onClick={() => navigate({ to: '/webos/livetv' } as never)}
              />
            )}
          </div>
        )}

        {!hasLibraries && (
          <div className="text-center text-neutral-500 py-12 mb-8">
            <p className="text-lg mb-2">Welcome to ViewRA</p>
            <p className="text-sm">Add a library to get started</p>
          </div>
        )}

        {/* Continue Watching */}
        {continueItems.length > 0 && (
          <>
            <div data-row className="mb-2">
              <SectionHeader title="Continue Watching" />
            </div>
            <div data-row className="grid grid-cols-5 gap-2 mb-8">
              {continueItems.slice(0, 5).map((item, index) => (
                <GridCard
                  key={`cw-${item.entity_type}-${item.entity_id}`}
                  id={index === 0 && !hasLibraries ? 'connect' : undefined}
                  mediaId={item.entity_id}
                  mediaType={entityTypeToMediaType(item.entity_type) as 'movie' | 'tv-show'}
                  title={item.title || 'Untitled'}
                  subtitle={item.episode_context
                    ? `S${item.episode_context.season} E${item.episode_context.episode}`
                    : undefined}
                  progress={item.progress?.percent}
                  onClick={() => handleContinueItemClick(item)}
                />
              ))}
            </div>
          </>
        )}

        {/* Up Next */}
        {upNextItems.length > 0 && (
          <>
            <div data-row className="mb-2">
              <SectionHeader title="Up Next" />
            </div>
            <div data-row className="grid grid-cols-5 gap-2 mb-8">
              {upNextItems.slice(0, 5).map((item, index) => (
                <GridCard
                  key={`un-${item.entity_type}-${item.entity_id}`}
                  id={index === 0 && !continueItems.length && !hasLibraries ? 'connect' : undefined}
                  mediaId={item.entity_id}
                  mediaType={entityTypeToMediaType(item.entity_type) as 'movie' | 'tv-show'}
                  title={item.title || 'Untitled'}
                  subtitle={item.episode_context
                    ? `S${item.episode_context.season} E${item.episode_context.episode}`
                    : undefined}
                  onClick={() => handleContinueItemClick(item)}
                />
              ))}
            </div>
          </>
        )}

        {/* Recommended */}
        {recommendedSection && (
          <>
            <div data-row className="mb-2">
              <SectionHeader title="Recommended for You" />
            </div>
            <div data-row className="grid grid-cols-5 gap-2 mb-8">
              {recommendedSection.data && (() => {
                const data = recommendedSection.data as MediaRowData
                const items = data.items ?? []
                return items.slice(0, 5).map((item, index) => (
                  <GridCard
                    key={`rec-${item.entity_type}-${item.entity_id}`}
                    id={index === 0 && !continueItems.length ? 'connect' : undefined}
                    mediaId={item.entity_id}
                    mediaType={entityTypeToMediaType(item.entity_type) as 'movie' | 'tv-show'}
                    title={item.title || 'Untitled'}
                    onClick={() => handleEntityNavigate(item.entity_type, item.entity_id)}
                  />
                ))
              })()}
            </div>
          </>
        )}

        {/* Unwatched Movies */}
        {unwatchedSection && (
          <>
            <div data-row className="mb-2">
              <SectionHeader title="Unwatched Movies" />
            </div>
            <div data-row className="grid grid-cols-5 gap-2 mb-8">
              {unwatchedSection.data && (() => {
                const data = unwatchedSection.data as MediaRowData
                const items = (data.movies ?? []).slice(0, 5).map(m => ({
                  entity_type: 'movie' as const,
                  entity_id: m.id,
                  title: m.title || 'Untitled',
                }))
                return items.map((item, index) => (
                  <GridCard
                    key={`uw-${item.entity_id}`}
                    id={index === 0 && !continueItems.length && !recommendedSection ? 'connect' : undefined}
                    mediaId={item.entity_id}
                    mediaType="movie"
                    title={item.title}
                    onClick={() => handleEntityNavigate('movie', item.entity_id)}
                  />
                ))
              })()}
            </div>
          </>
        )}

        {/* New Additions */}
        {newAdditionsSection && (
          <>
            <div data-row className="mb-2">
              <SectionHeader title="Recently Added" />
            </div>
            <div data-row className="grid grid-cols-5 gap-2 mb-8">
              {newAdditionsSection.data && (() => {
                const data = newAdditionsSection.data as MediaRowData
                const movieItems = (data.movies ?? []).slice(0, 5)
                const showItems = (data.shows ?? []).slice(0, 5)
                const items: Array<{ entity_type: 'movie' | 'tv_show'; entity_id: number; title: string }> = [
                  ...movieItems.map(m => ({
                    entity_type: 'movie' as const,
                    entity_id: m.id,
                    title: m.title || 'Untitled',
                  })),
                  ...showItems.map(s => ({
                    entity_type: 'tv_show' as const,
                    entity_id: s.id ?? 0,
                    title: s.title || 'Untitled',
                  })),
                ].slice(0, 5)
                return items.map((item, index) => (
                  <GridCard
                    key={`new-${item.entity_type}-${item.entity_id}`}
                    id={index === 0 && !continueItems.length && !recommendedSection ? 'connect' : undefined}
                    mediaId={item.entity_id}
                    mediaType={entityTypeToMediaType(item.entity_type) as 'movie' | 'tv-show'}
                    title={item.title}
                    onClick={() => handleEntityNavigate(item.entity_type, item.entity_id)}
                  />
                ))
              })()}
            </div>
          </>
        )}
      </div>

      {/* Footer hint */}
      <div className="px-4 py-3 text-center text-neutral-500 text-xs">
        <p>ViewRA &mdash; Use arrow keys to navigate</p>
      </div>
    </div>
  )
}
