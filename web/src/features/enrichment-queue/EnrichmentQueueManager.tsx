import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Loading, Alert, Button } from '@/components/ui'
import { text } from '@/styles/semantic'
import { cn } from '@/lib/utils'
import {
  useGetApiEnrichmentQueue,
  useGetApiEnrichmentStats,
  getGetApiEnrichmentQueueQueryKey,
  postApiEnrichmentPause,
  postApiEnrichmentResume,
  postApiEnrichmentRetry,
} from '@/lib/api/generated/enrichment/enrichment'
import type { InternalApiHandlersEnrichmentQueueItem } from '@/lib/api/generated/models'
import {
  Play,
  Pause,
  RefreshCw,
  RotateCcw,
  ChevronLeft,
  ChevronRight,
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
} from 'lucide-react'

const PAGE_SIZE = 50

const STATUS_FILTERS = [
  { value: '', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'processing', label: 'Processing' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
] as const

const statusColor = (status: string) => {
  switch (status) {
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'processing':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    case 'completed':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'failed':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'skipped':
      return 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-400'
    default:
      return 'bg-neutral-100 text-neutral-800 dark:bg-neutral-800 dark:text-neutral-400'
  }
}

const statusIcon = (status: string) => {
  switch (status) {
    case 'pending':
      return <Clock className="w-3 h-3" />
    case 'processing':
      return <Loader2 className="w-3 h-3 animate-spin" />
    case 'completed':
      return <CheckCircle2 className="w-3 h-3" />
    case 'failed':
      return <XCircle className="w-3 h-3" />
    default:
      return null
  }
}

export const EnrichmentQueueManager = () => {
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(0)

  const {
    data: statsData,
    isLoading: statsLoading,
  } = useGetApiEnrichmentStats({ query: { refetchInterval: 5000 } })

  const {
    data: queueData,
    isLoading: queueLoading,
    error: queueError,
    isFetching,
  } = useGetApiEnrichmentQueue(
    { status: statusFilter || undefined, limit: PAGE_SIZE, offset: page * PAGE_SIZE },
    { query: { refetchInterval: 5000 } }
  )

  const stats = statsData?.status === 200 ? statsData.data : {}
  const items = queueData?.status === 200 ? queueData.data.items || [] : []
  const total = queueData?.status === 200 ? queueData.data.total || 0 : 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const totals = Object.values(stats).reduce(
    (acc, s) => ({
      pending: acc.pending + (s.pendingCount || 0),
      processing: acc.processing + (s.processingCount || 0),
      completed: acc.completed + (s.completedCount || 0),
      failed: acc.failed + (s.failedCount || 0),
    }),
    { pending: 0, processing: 0, completed: 0, failed: 0 }
  )

  const handlePause = async () => {
    await postApiEnrichmentPause()
    queryClient.invalidateQueries({ queryKey: ['getApiEnrichmentStatus'] })
    queryClient.invalidateQueries({ queryKey: ['getApiEnrichmentStats'] })
  }

  const handleResume = async () => {
    await postApiEnrichmentResume()
    queryClient.invalidateQueries({ queryKey: ['getApiEnrichmentStatus'] })
    queryClient.invalidateQueries({ queryKey: ['getApiEnrichmentStats'] })
  }

  const handleRetry = async (jobId: number) => {
    await postApiEnrichmentRetry({ job_id: jobId })
    queryClient.invalidateQueries({
      queryKey: getGetApiEnrichmentQueueQueryKey({
        status: statusFilter || undefined,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      }),
    })
  }

  const handleFilterChange = (filter: string) => {
    setStatusFilter(filter)
    setPage(0)
  }

  if (statsLoading) {
    return <Loading text="Loading enrichment queue..." />
  }

  return (
    <div className="space-y-6">
      {/* Stats Bar */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <StatCard
          label="Pending"
          value={totals.pending}
          icon={<Clock className="w-4 h-4" />}
          color="text-yellow-600 dark:text-yellow-400"
        />
        <StatCard
          label="Processing"
          value={totals.processing}
          icon={<Loader2 className="w-4 h-4 animate-spin" />}
          color="text-blue-600 dark:text-blue-400"
        />
        <StatCard
          label="Completed"
          value={totals.completed}
          icon={<CheckCircle2 className="w-4 h-4" />}
          color="text-green-600 dark:text-green-400"
        />
        <StatCard
          label="Failed"
          value={totals.failed}
          icon={<XCircle className="w-4 h-4" />}
          color="text-red-600 dark:text-red-400"
        />
      </div>

      {/* Control Bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button variant="secondary" size="sm" onClick={handlePause}>
            <Pause className="w-4 h-4 mr-1.5" />
            Pause
          </Button>
          <Button variant="secondary" size="sm" onClick={handleResume}>
            <Play className="w-4 h-4 mr-1.5" />
            Resume
          </Button>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() =>
            queryClient.invalidateQueries({
              queryKey: getGetApiEnrichmentQueueQueryKey({
                status: statusFilter || undefined,
                limit: PAGE_SIZE,
                offset: page * PAGE_SIZE,
              }),
            })
          }
          disabled={isFetching}
        >
          <RefreshCw
            className={cn('w-4 h-4 mr-1.5', isFetching && 'animate-spin')}
          />
          Refresh
        </Button>
      </div>

      {/* Filter Tabs */}
      <div className="flex items-center gap-1 border-b border-neutral-200 dark:border-neutral-800">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            onClick={() => handleFilterChange(f.value)}
            className={cn(
              'px-3 py-2 text-sm font-medium transition-colors',
              'border-b-2 -mb-px',
              statusFilter === f.value
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : 'border-transparent text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200'
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Error */}
      {queueError && (
        <Alert variant="error">Failed to load enrichment queue.</Alert>
      )}

      {/* Queue Table */}
      {queueLoading ? (
        <Loading text="Loading queue items..." />
      ) : items.length === 0 ? (
        <div className={cn('text-center py-12', text.secondary)}>
          <p className="text-sm">No queue items found.</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr
                className={cn(
                  'border-b border-neutral-200 dark:border-neutral-800',
                  text.secondary
                )}
              >
                <th className="text-left py-2 px-3 font-medium">Title</th>
                <th className="text-left py-2 px-3 font-medium">Type</th>
                <th className="text-left py-2 px-3 font-medium">Stage</th>
                <th className="text-right py-2 px-3 font-medium">Priority</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-right py-2 px-3 font-medium">Attempts</th>
                <th className="text-right py-2 px-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item: InternalApiHandlersEnrichmentQueueItem) => (
                <tr
                  key={item.id}
                  className="border-b border-neutral-100 dark:border-neutral-900"
                >
                  <td className="py-2 px-3 max-w-[300px] truncate">
                    {item.title || (
                      <span className={text.tertiary}>Unknown</span>
                    )}
                  </td>
                  <td className={cn('py-2 px-3', text.secondary)}>
                    {item.media_type}
                  </td>
                  <td className={cn('py-2 px-3', text.secondary)}>
                    {item.stage}
                  </td>
                  <td className={cn('py-2 px-3 text-right', text.secondary)}>
                    {item.priority}
                  </td>
                  <td className="py-2 px-3">
                    <span
                      className={cn(
                        'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
                        statusColor(item.status || '')
                      )}
                    >
                      {statusIcon(item.status || '')}
                      {item.status}
                    </span>
                  </td>
                  <td
                    className={cn(
                      'py-2 px-3 text-right',
                      text.secondary
                    )}
                  >
                    {item.attempts}/{item.max_attempts}
                  </td>
                  <td className="py-2 px-3 text-right">
                    {item.status === 'failed' &&
                      typeof item.id === 'number' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRetry(item.id as number)}
                        >
                          <RotateCcw className="w-3 h-3 mr-1" />
                          Retry
                        </Button>
                      )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className={cn('text-sm', text.secondary)}>
            Page {page + 1} of {totalPages} ({total} items)
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
            >
              <ChevronLeft className="w-4 h-4" />
              Previous
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
            >
              Next
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

const StatCard = ({
  label,
  value,
  icon,
  color,
}: {
  label: string
  value: number
  icon: React.ReactNode
  color: string
}) => (
  <div
    className={cn(
      'p-4 rounded-xl',
      'bg-white/50 dark:bg-white/[0.02]',
      'border border-neutral-200/50 dark:border-white/10'
    )}
  >
    <div className="flex items-center gap-2 mb-1">
      <span className={color}>{icon}</span>
      <span className={cn('text-xs font-medium uppercase tracking-wide', text.secondary)}>
        {label}
      </span>
    </div>
    <p className={cn('text-2xl font-bold', text.primary)}>{value}</p>
  </div>
)
