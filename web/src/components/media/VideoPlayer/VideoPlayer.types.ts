import type { MediaMetadata } from '@/lib/types/video'
import type { QualityOption, SavedPreferences } from '@/lib/hooks/useMediaPlayback'

export interface VideoPlayerProps {
  mediaId: number
  streamUrl: string
  initialPosition?: number // in seconds
  duration?: number // in seconds
  metadata?: MediaMetadata
  onClose?: () => void
  onTimeUpdate?: (time: number) => void // Called periodically with current playback position
  /** Available quality options from backend (filtered by source resolution) */
  availableQualities?: QualityOption[]
  /** Currently selected quality ID */
  selectedQualityId?: string | null
  /** Callback to change quality - rebuilds URL and reloads stream from current position */
  onQualityChange?: (qualityId: string, currentPosition: number) => Promise<void>
  /** Saved playback preferences from previous session */
  savedPreferences?: SavedPreferences | null
  /** Info about the next episode (for auto-play overlay) */
  nextEpisodeInfo?: { title: string; season: number; episode: number; episodeTitle?: string }
  /** Called when next episode should start (auto-play) */
  onAutoPlayNext?: () => void
  /** Called when user cancels auto-play */
  onAutoPlayCancel?: () => void
  /** Navigate to next episode */
  onPlayNext?: () => void
  /** Navigate to previous episode */
  onPlayPrev?: () => void
}
