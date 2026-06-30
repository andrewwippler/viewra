import { useState, useMemo } from 'react'
import { Button, ConfirmDialog } from '@/components/ui'
import { useAuth } from '@/contexts/AuthContext'
import { useNitpickyEditsAvailable, useScanMissing, useRemoveMissing } from '../hooks/useNitpickyEdits'
import type { MissingItem } from '../api/client'

type SelectionMap = Record<string, boolean>

export const MissingFilesManager = () => {
  const { user } = useAuth()
  const { data: plugin } = useNitpickyEditsAvailable()
  const scan = useScanMissing()
  const remove = useRemoveMissing()

  const [selected, setSelected] = useState<SelectionMap>({})
  const [showConfirm, setShowConfirm] = useState(false)

  const selectedCount = useMemo(
    () => Object.values(selected).filter(Boolean).length,
    [selected]
  )

  const isAdmin = user?.is_admin ?? false
  const needsAdmin = plugin?.editModeRequired ?? true

  if (!plugin?.available || (needsAdmin && !isAdmin)) {
    return (
      <div className="text-neutral-400 text-sm py-8 text-center">
        Nitpicky Edits plugin is not available or you lack admin permissions.
      </div>
    )
  }

  const items = scan.data ?? []
  const hasResults = scan.isSuccess && items.length > 0
  const noResults = scan.isSuccess && items.length === 0
  const allSelected = hasResults && selectedCount === items.length

  const handleScan = () => {
    setSelected({})
    scan.refetch()
  }

  const toggleAll = () => {
    if (allSelected) {
      setSelected({})
    } else {
      const all: SelectionMap = {}
      for (const item of items) {
        const key = item.item_type === 'tv_show' ? `show-${item.show_id}` : `media-${item.media_id}`
        all[key] = true
      }
      setSelected(all)
    }
  }

  const toggleItem = (key: string) => {
    setSelected((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const handleRemove = async () => {
    const mediaIds: number[] = []
    const showIds: number[] = []

    for (const item of items) {
      const key = item.item_type === 'tv_show' ? `show-${item.show_id}` : `media-${item.media_id}`
      if (selected[key]) {
        if (item.item_type === 'tv_show' && item.show_id) {
          showIds.push(item.show_id)
        } else if (item.media_id) {
          mediaIds.push(item.media_id)
        }
      }
    }

    setShowConfirm(false)

    remove.mutate(
      { media_ids: mediaIds, show_ids: showIds },
      { onSuccess: () => setSelected({}) }
    )
  }

  const getItemKey = (item: MissingItem) =>
    item.item_type === 'tv_show' ? `show-${item.show_id}` : `media-${item.media_id}`

  const getItemLabel = (item: MissingItem) => {
    if (item.item_type === 'tv_show') {
      return `${item.title} (Show Directory)`
    }
    if (item.item_type === 'tv_episode') {
      return `${item.show_title} - S${item.season_number ?? '?'}:E${item.episode_number ?? '?'} - ${item.title}`
    }
    return item.title
  }

  const getItemSubtitle = (item: MissingItem) => {
    if (item.item_type === 'tv_show') {
      return 'Missing show directory on disk'
    }
    if (item.item_type === 'tv_episode') {
      return item.library_name ? `TV Episodes • ${item.library_name}` : 'TV Episode'
    }
    return item.library_name ? `Movies • ${item.library_name}` : 'Movie'
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold text-neutral-900 dark:text-white mb-1">
          Storage Cleanup
        </h2>
        <p className="text-sm text-neutral-500 dark:text-neutral-400">
          Scan your media directories to detect files that have been moved or deleted.
          Missing items can be removed from the database.
        </p>
      </div>

      <div className="flex items-center gap-3">
        <Button
          onClick={handleScan}
          disabled={scan.isFetching}
          variant="primary"
        >
          {scan.isFetching ? 'Scanning...' : 'Scan for Missing Files'}
        </Button>
        {scan.isFetching && (
          <div className="w-5 h-5 rounded-full animate-spin border-2 border-primary-500 border-t-transparent" />
        )}
      </div>

      {scan.isError && (
        <div className="p-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-300 text-sm">
          Scan failed: {(scan.error as Error)?.message || 'Unknown error'}
        </div>
      )}

      {noResults && (
        <div className="p-8 text-center text-neutral-400 text-sm border border-neutral-200 dark:border-white/10 rounded-lg">
          All media files are present on disk. No missing items found.
        </div>
      )}

      {hasResults && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-sm text-neutral-500 dark:text-neutral-400">
              Found {items.length} missing item{items.length > 1 ? 's' : ''}
            </p>
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 text-sm text-neutral-600 dark:text-neutral-300 cursor-pointer">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleAll}
                  className="rounded border-neutral-300 dark:border-neutral-600"
                />
                {allSelected ? 'Deselect All' : 'Select All'}
              </label>
              <Button
                onClick={() => setShowConfirm(true)}
                disabled={selectedCount === 0 || remove.isPending}
                variant="danger"
                size="sm"
              >
                {remove.isPending
                  ? 'Removing...'
                  : `Remove Selected${selectedCount > 0 ? ` (${selectedCount})` : ''}`}
              </Button>
            </div>
          </div>

          <div className="border border-neutral-200 dark:border-white/10 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-neutral-50 dark:bg-neutral-900/50 border-b border-neutral-200 dark:border-white/10">
                  <th className="w-10 p-3 text-left" />
                  <th className="p-3 text-left font-medium text-neutral-500 dark:text-neutral-400">
                    Title
                  </th>
                  <th className="p-3 text-left font-medium text-neutral-500 dark:text-neutral-400 w-28">
                    Type
                  </th>
                  <th className="p-3 text-left font-medium text-neutral-500 dark:text-neutral-400">
                    File Path
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => {
                  const key = getItemKey(item)
                  return (
                    <tr
                      key={key}
                      className="border-b border-neutral-100 dark:border-white/5 hover:bg-neutral-50 dark:hover:bg-white/5 transition-colors"
                    >
                      <td className="p-3 text-center">
                        <input
                          type="checkbox"
                          checked={!!selected[key]}
                          onChange={() => toggleItem(key)}
                          className="rounded border-neutral-300 dark:border-neutral-600"
                        />
                      </td>
                      <td className="p-3">
                        <div className="font-medium text-neutral-900 dark:text-white">
                          {getItemLabel(item)}
                        </div>
                        <div className="text-xs text-neutral-400 mt-0.5">
                          {getItemSubtitle(item)}
                        </div>
                      </td>
                      <td className="p-3">
                        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
                          item.item_type === 'tv_show'
                            ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
                            : item.item_type === 'tv_episode'
                              ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                              : 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300'
                        }`}>
                          {item.item_type === 'tv_show' ? 'Show' : item.item_type === 'tv_episode' ? 'Episode' : 'Movie'}
                        </span>
                      </td>
                      <td className="p-3 text-neutral-500 dark:text-neutral-400 font-mono text-xs truncate max-w-xs">
                        {item.file_path}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <ConfirmDialog
        isOpen={showConfirm}
        title="Remove Missing Items"
        message={`Are you sure you want to remove ${selectedCount} missing item${selectedCount > 1 ? 's' : ''} from the database? The operation cannot be undone.`}
        confirmText={remove.isPending ? 'Removing...' : 'Remove'}
        variant="danger"
        onConfirm={handleRemove}
        onCancel={() => setShowConfirm(false)}
      />
    </div>
  )
}
