import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Button, ConfirmDialog } from '@/components/ui'
import { IdentifyModal } from './IdentifyModal'
import { useNitpickyEditsAvailable, useIdentifyMovie, useIdentifyTVShow, useDeleteShow, useMarkWatchedMedia, useMarkUnwatchedMedia, useMarkSeasonWatched, useMarkSeasonUnwatched } from '../hooks/useNitpickyEdits'
import { mediaApi } from '@/lib/api'
import { useAuth } from '@/contexts/AuthContext'

interface AdminActionsProps {
  mediaType: 'movie' | 'tv'
  mediaId: number
  mediaTitle: string
  seasonId?: number
  episodeId?: number
  isWatched?: boolean
  onDeleteNavigate?: string
  onAction?: () => void
}

export const AdminActions = ({ mediaType, mediaId, mediaTitle, seasonId, episodeId, isWatched, onDeleteNavigate, onAction }: AdminActionsProps) => {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { data: plugin } = useNitpickyEditsAvailable()
  const identifyMovie = useIdentifyMovie()
  const identifyTVShow = useIdentifyTVShow()
  const deleteShow = useDeleteShow()
  const markWatched = useMarkWatchedMedia()
  const markUnwatched = useMarkUnwatchedMedia()
  const markSeasonWatched = useMarkSeasonWatched()
  const markSeasonUnwatched = useMarkSeasonUnwatched()

  const [isOpen, setIsOpen] = useState(false)
  const [showIdentifyModal, setShowIdentifyModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  if (!plugin?.available) {
    return null
  }

  const isAdmin = user?.is_admin ?? false
  const needsAdmin = plugin.editModeRequired
  if (needsAdmin && !isAdmin) {
    return null
  }

  const handleIdentify = (data: { imdb_id: string }) => {
    if (mediaType === 'movie') {
      identifyMovie.mutate(
        { id: mediaId, data },
        { onSuccess: () => { setShowIdentifyModal(false); setIsOpen(false); onAction?.() } }
      )
    } else {
      identifyTVShow.mutate(
        { id: mediaId, data },
        { onSuccess: () => { setShowIdentifyModal(false); setIsOpen(false); onAction?.() } }
      )
    }
  }

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      if (mediaType === 'tv') {
        await new Promise<void>((resolve, reject) => {
          deleteShow.mutate(mediaId, {
            onSuccess: () => resolve(),
            onError: (err) => reject(err),
          })
        })
      } else {
        await mediaApi.deleteMedia(mediaId)
      }
      setShowDeleteConfirm(false)
      setIsOpen(false)
      if (onDeleteNavigate) {
        navigate({ to: onDeleteNavigate })
      }
      onAction?.()
    } catch {
      // Delete failed — dialog stays open
    } finally {
      setIsDeleting(false)
    }
  }

  const handleToggleWatched = () => {
    if (isWatched) {
      markUnwatched.mutate(mediaId, {
        onSuccess: () => { setIsOpen(false); onAction?.() }
      })
    } else {
      markWatched.mutate(mediaId, {
        onSuccess: () => { setIsOpen(false); onAction?.() }
      })
    }
  }

  return (
    <>
      <Button
        onClick={() => setIsOpen(!isOpen)}
        variant="secondary"
        size="sm"
      >
        {isOpen ? 'Close' : 'Edit'}
      </Button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 z-50 w-56 rounded-lg border border-neutral-200 dark:border-white/10 bg-white dark:bg-neutral-900 shadow-xl">
          <div className="py-1">
            <button
              onClick={() => setShowIdentifyModal(true)}
              className="w-full text-left px-4 py-2 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
            >
              Identify
            </button>

            {mediaType === 'movie' && (
              <>
                <button
                  onClick={handleToggleWatched}
                  className="w-full text-left px-4 py-2 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  {isWatched ? 'Mark Unwatched' : 'Mark Watched'}
                </button>
                <div className="border-t border-neutral-200 dark:border-white/10 my-1" />
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  Delete
                </button>
              </>
            )}

            {mediaType === 'tv' && !episodeId && !seasonId && (
              <>
                <button
                  onClick={handleToggleWatched}
                  className="w-full text-left px-4 py-2 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  {isWatched ? 'Mark Unwatched' : 'Mark Watched'}
                </button>
                <div className="border-t border-neutral-200 dark:border-white/10 my-1" />
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  Delete Show
                </button>
              </>
            )}

            {seasonId && (
              <>
                <button
                  onClick={() => markSeasonWatched.mutate(seasonId, {
                    onSuccess: () => { setIsOpen(false); onAction?.() }
                  })}
                  className="w-full text-left px-4 py-2 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  Mark Season Watched
                </button>
                <button
                  onClick={() => markSeasonUnwatched.mutate(seasonId, {
                    onSuccess: () => { setIsOpen(false); onAction?.() }
                  })}
                  className="w-full text-left px-4 py-2 text-sm text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  Mark Season Unwatched
                </button>
                <div className="border-t border-neutral-200 dark:border-white/10 my-1" />
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-neutral-100 dark:hover:bg-white/5 transition-colors"
                >
                  Delete Show
                </button>
              </>
            )}
          </div>
        </div>
      )}

      <IdentifyModal
        isOpen={showIdentifyModal}
        onClose={() => setShowIdentifyModal(false)}
        mediaType={mediaType}
        mediaTitle={mediaTitle}
        onIdentify={handleIdentify}
        isPending={identifyMovie.isPending || identifyTVShow.isPending}
      />

      <ConfirmDialog
        isOpen={showDeleteConfirm}
        title={mediaType === 'tv' ? 'Delete TV Show' : 'Remove Movie'}
        message={mediaType === 'tv'
          ? `Are you sure you want to remove "${mediaTitle}" and all its episodes from your library? The underlying files on disk will not be deleted.`
          : `Are you sure you want to remove "${mediaTitle}" from your library? The underlying files on disk will not be deleted.`
        }
        confirmText={isDeleting ? 'Removing...' : 'Remove'}
        variant="danger"
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteConfirm(false)}
      />
    </>
  )
}
