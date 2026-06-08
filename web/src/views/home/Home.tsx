import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { Library, Film, Tv, Music, RefreshCw, AlertCircle } from 'lucide-react'
import { useCallback } from 'react'
import { SearchHero, WidgetSection, WidgetLocation, HeroBackdrop } from '@/components/home'
import { useSearchHeroData, useHomeSections, BatchImagesProvider, BatchProgressProvider } from '@/lib/hooks'
import { cn } from '@/lib/utils'
import { progressApi } from '@/lib/api/progress'
import type { MediaRowData, ContinueWatchingData, ContinueWatchingItem } from '@/components/home/widgets/widget.types'

/**
 * Home - Main landing page with search and widget-based content rows
 *
 * Features:
 * - Search hero with contextual suggestions
 * - Dynamic content rows via widget system (continue watching, recently added, favorites, etc.)
 * - Loading skeleton while fetching
 * - Empty state for new users
 * - Error state with retry
 */
export const Home = () => {
  const { data: searchHeroData, isLoading: isLoadingSearch } = useSearchHeroData()
  const { data: homeSections, isLoading: isLoadingSections, error, refetch } = useHomeSections()

  // Sections for the main content area
  const sections = homeSections?.sections ?? []

  // Extract hero data from response metadata
  const heroData = homeSections?.meta?.hero

  // Handle removing an item from continue watching
  const handleRemoveContinueWatching = useCallback(async (item: ContinueWatchingItem) => {
    const mediaId = item.entity_type === 'tv_show' && item.episode_context?.episode_media_id
      ? item.episode_context.episode_media_id
      : item.entity_id
    try {
      await progressApi.deleteProgress(mediaId)
      refetch()
    } catch {
      // Silently fail - the item might already be gone on next refetch
    }
  }, [refetch])

  // Extract all media IDs and entity IDs from sections for batch loading
  const { mediaIds, tvShowEntityIds } = useMemo(() => {
    const media: number[] = []
    const tvShows: number[] = []
    
    for (const section of sections) {
      // Handle MediaRowData (movies, shows, and recommendation items)
      const mediaData = section.data as MediaRowData
      if (mediaData?.movies) {
        media.push(...mediaData.movies.map((m) => m.id))
      }
      if (mediaData?.shows) {
        // TV shows are entities, not media items - use entity IDs
        tvShows.push(...mediaData.shows.filter((s) => s.id !== null && s.id !== undefined).map((s) => s.id as number))
      }
      if (mediaData?.items) {
        // Separate recommendation items by entity type
        for (const item of mediaData.items) {
          if (item.entity_type === 'tv_show') {
            tvShows.push(item.entity_id)
          } else {
            // Movies and other types use media IDs
            media.push(item.entity_id)
          }
        }
      }

      // Handle ContinueWatchingData with items array
      const continueData = section.data as ContinueWatchingData
      if (continueData?.items) {
        for (const item of continueData.items) {
          if (item.entity_type === 'tv_show' && item.episode_context?.episode_media_id) {
            // TV episodes use episode media ID
            media.push(item.episode_context.episode_media_id)
          } else if (item.entity_type === 'tv_show') {
            tvShows.push(item.entity_id)
          } else {
            media.push(item.entity_id)
          }
        }
      }
    }
    return { mediaIds: media, tvShowEntityIds: tvShows }
  }, [sections])

  // Show loading skeleton
  if (isLoadingSections) {
    return <HomeLoadingSkeleton />
  }

  // Show error state
  if (error) {
    return <HomeErrorState onRetry={refetch} />
  }

  // Show empty state if no sections
  const hasContent = sections.length > 0
  if (!hasContent) {
    return <HomeEmptyState />
  }

  return (
    <BatchImagesProvider mediaIds={mediaIds} entityIds={tvShowEntityIds} mediaType="tv_show">
      <BatchProgressProvider mediaIds={mediaIds}>
        <div className="relative h-full overflow-auto">
          {/* Hero Backdrop */}
          {heroData?.backdrop_media_id && heroData?.backdrop_media_type && (
            <HeroBackdrop
              mediaId={heroData.backdrop_media_id}
              mediaType={heroData.backdrop_media_type}
              greeting={heroData.greeting}
              dateText={heroData.date_text}
            />
          )}

          <div className="relative z-10 page-enter">
            {/* Search Hero */}
            {!isLoadingSearch && searchHeroData && (
              <div className={cn('p-8 pb-4', heroData?.backdrop_media_id && 'pt-24')}>
                <SearchHero data={searchHeroData} />
              </div>
            )}

            {/* Content Sections */}
            <div className="p-8 pt-4 space-y-10">
              {/* All widget sections (continue watching, recently added, favorites, etc.) */}
              <WidgetSection
                sections={sections}
                location={WidgetLocation.HomepageSections}
                onRemoveContinueWatching={handleRemoveContinueWatching}
              />
            </div>
          </div>
        </div>
      </BatchProgressProvider>
    </BatchImagesProvider>
  )
}

/**
 * Loading skeleton for the home page
 */
const HomeLoadingSkeleton = () => {
  return (
    <div className="h-full overflow-auto">
      <div className="p-8">
        {/* Search bar skeleton */}
        <div className="max-w-2xl mx-auto mb-8">
          <div className="h-12 bg-neutral-200 dark:bg-neutral-800 rounded-xl skeleton-shimmer" />
          <div className="flex gap-2 mt-4 justify-center">
            {[1, 2, 3, 4, 5].map((i) => (
              <div
                key={i}
                className="h-8 w-24 bg-neutral-200 dark:bg-neutral-800 rounded-full skeleton-shimmer"
              />
            ))}
          </div>
        </div>

        {/* Row skeletons */}
        {[1, 2, 3].map((row) => (
          <div key={row} className="mb-10">
            <div className="h-6 w-40 bg-neutral-200 dark:bg-neutral-800 rounded mb-5 skeleton-shimmer" />
            <div className="flex gap-4">
              {[1, 2, 3, 4, 5, 6].map((card) => (
                <div key={card} className="w-48 shrink-0">
                  <div className="aspect-[2/3] bg-neutral-200 dark:bg-neutral-800 rounded-xl skeleton-shimmer" />
                  <div className="mt-3 space-y-2">
                    <div className="h-4 bg-neutral-200 dark:bg-neutral-800 rounded skeleton-shimmer" />
                    <div className="h-3 w-2/3 bg-neutral-200 dark:bg-neutral-800 rounded skeleton-shimmer" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

/**
 * Empty state shown when there's no content
 */
const HomeEmptyState = () => {
  return (
    <div className="h-full overflow-auto">
      <div className="min-h-full flex items-center justify-center p-8">
        <div className="max-w-md text-center">
          <div className="mb-6 flex justify-center gap-3">
            <div className="w-12 h-12 rounded-xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <Film className="w-6 h-6 text-primary-600 dark:text-primary-400" />
            </div>
            <div className="w-12 h-12 rounded-xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <Tv className="w-6 h-6 text-primary-600 dark:text-primary-400" />
            </div>
            <div className="w-12 h-12 rounded-xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
              <Music className="w-6 h-6 text-primary-600 dark:text-primary-400" />
            </div>
          </div>

          <h2 className="text-xl font-semibold text-neutral-900 dark:text-white mb-2">
            Welcome to ViewRA
          </h2>
          <p className="text-neutral-600 dark:text-neutral-400 mb-6">
            Get started by adding a library to see your movies, TV shows, and music here.
          </p>

          <Link
            to="/libraries"
            className={cn(
              'inline-flex items-center gap-2 px-5 py-2.5 rounded-lg',
              'bg-primary-600 hover:bg-primary-700',
              'text-white font-medium',
              'transition-colors'
            )}
          >
            <Library className="w-4 h-4" />
            Add Library
          </Link>
        </div>
      </div>
    </div>
  )
}

/**
 * Error state with retry button
 */
const HomeErrorState = ({ onRetry }: { onRetry: () => void }) => {
  return (
    <div className="h-full overflow-auto">
      <div className="min-h-full flex items-center justify-center p-8">
        <div className="max-w-md text-center">
          <div className="mb-6 flex justify-center">
            <div className="w-14 h-14 rounded-full bg-error-100 dark:bg-error-900/30 flex items-center justify-center">
              <AlertCircle className="w-7 h-7 text-error-600 dark:text-error-400" />
            </div>
          </div>

          <h2 className="text-xl font-semibold text-neutral-900 dark:text-white mb-2">
            Unable to load content
          </h2>
          <p className="text-neutral-600 dark:text-neutral-400 mb-6">
            Something went wrong while loading your home screen. Please try again.
          </p>

          <button
            onClick={onRetry}
            className={cn(
              'inline-flex items-center gap-2 px-5 py-2.5 rounded-lg cursor-pointer',
              'bg-neutral-200 dark:bg-neutral-800',
              'hover:bg-neutral-300 dark:hover:bg-neutral-700',
              'text-neutral-900 dark:text-white font-medium',
              'transition-colors'
            )}
          >
            <RefreshCw className="w-4 h-4" />
            Try Again
          </button>
        </div>
      </div>
    </div>
  )
}
