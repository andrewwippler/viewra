/**
 * TVMediaCard Component
 * 
 * A media card optimized for TV remote control navigation.
 * Features:
 * - High-contrast focus states for 10-foot viewing
 * - Large touch targets (48px minimum)
 * - Smooth scale animation on focus
 * - webOS-optimized styling
 */

import { useCallback } from 'react'
import { cn } from '@/lib/utils'

interface TVMediaCardProps {
  /** Unique identifier for the card */
  id: string | number
  /** Media title */
  title: string
  /** Year of release */
  year?: number
  /** Poster image URL */
  posterUrl?: string
  /** Overlay content (e.g., progress bar) */
  overlay?: React.ReactNode
  /** Whether the media is fully watched */
  isWatched?: boolean
  /** Additional CSS classes */
  className?: string
  /** Click handler */
  onClick?: () => void
  /** Hover handler (for mouse) */
  onHover?: () => void
}

/**
 * TV-optimized media card with enhanced focus states
 */
export const TVMediaCard = ({
  id,
  title,
  year,
  posterUrl,
  overlay,
  isWatched,
  className,
  onClick,
  onHover,
}: TVMediaCardProps) => {
  const handleClick = useCallback(() => {
    onClick?.()
  }, [onClick])

  const handleMouseEnter = useCallback(() => {
    onHover?.()
  }, [onHover])

  return (
    <div
      className={cn(
        'focusable', // Enable spatial navigation
        'relative',
        'rounded-xl',
        'overflow-hidden',
        'cursor-pointer',
        'min-w-[160px]', // Minimum width for TV
        'min-h-[200px]', // Minimum height for TV
        'transition-transform duration-150 ease-out',
        'bg-neutral-800',
        className
      )}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      tabIndex={0}
      role="button"
      aria-label={`${title}${year ? ` (${year})` : ''}`}
      data-media-id={id}
    >
      {/* Poster Image */}
      <div className="aspect-[2/3] w-full bg-neutral-700">
        {posterUrl ? (
          <img
            src={posterUrl}
            alt={`${title} poster`}
            width="2"
            height="3"
            className="w-full h-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <span className="text-neutral-500 text-4xl">🎬</span>
          </div>
        )}
      </div>

      {/* Info Overlay */}
      <div className="absolute inset-x-0 bottom-0 p-3 bg-gradient-to-t from-black/90 to-transparent">
        <h3 className="text-white text-sm font-medium line-clamp-2">{title}</h3>
        {year && <span className="text-neutral-400 text-xs">{year}</span>}
      </div>

      {/* Watched badge */}
      {isWatched && (
        <div className="absolute top-2 right-2 z-10 w-7 h-7 rounded-full bg-green-500 flex items-center justify-center shadow-lg">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </div>
      )}

      {/* Progress/Status Overlay */}
      {overlay && (
        <div className="absolute inset-0 flex items-end p-2">
          {overlay}
        </div>
      )}

      {/* Focus Ring - Custom styled for TV */}
      <div
        className={cn(
          'absolute inset-0',
          'rounded-xl',
          'pointer-events-none',
          'transition-opacity duration-150'
        )}
        style={{
          boxShadow: 'inset 0 0 0 0px rgba(207, 16, 32, 0)',
        }}
      />
    </div>
  )
}

/**
 * TVMediaCard with Progress indicator
 */
interface TVMediaCardWithProgressProps extends TVMediaCardProps {
  progress?: number // 0-100
}

export const TVMediaCardWithProgress = ({
  progress,
  ...props
}: TVMediaCardWithProgressProps) => {
  return (
    <TVMediaCard
      {...props}
      overlay={
        progress !== undefined && progress > 0 ? (
          <div className="w-full h-1 bg-neutral-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-red-600 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        ) : null
      }
    />
  )
}

/**
 * TV Grid Container
 * 
 * Provides a grid layout optimized for spatial navigation.
 * Items will snap to grid positions for consistent navigation.
 */
interface TVMediaGridProps {
  children: React.ReactNode
  /** Number of columns (default: 4) */
  columns?: number
  /** Gap between items */
  gap?: number
  className?: string
}

export const TVMediaGrid = ({
  children,
  columns = 4,
  gap = 24,
  className,
}: TVMediaGridProps) => {
  return (
    <div
      className={cn(
        'grid',
        'p-6',
        className
      )}
      style={{
        gridTemplateColumns: `repeat(${columns}, minmax(160px, 1fr))`,
        gap: `${gap}px`,
      }}
    >
      {children}
    </div>
  )
}