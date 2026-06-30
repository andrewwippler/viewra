import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router'
import { useEffect, useState, useCallback } from 'react'
import { LiveStreamPlayer } from '@/components/livetv/LiveStreamPlayer'
import { EpgInfoBox } from '@/components/livetv/EpgInfoBox'
import { fetchChannels, fetchEPG, type LiveChannel, type EpgProgram } from '@/lib/api/livetv'

interface LivetvPlaySearch {
  libraryId?: number
  channelId?: number
}

const LivetvPlay = () => {
  const navigate = useNavigate()
  const search = useSearch({ from: Route.id })
  const { libraryId, channelId } = search

  const [channel, setChannel] = useState<LiveChannel | null>(null)
  const [programs, setPrograms] = useState<EpgProgram[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Load channel and EPG data
  useEffect(() => {
    if (!libraryId || !channelId) {
      setError('Missing library or channel ID')
      setLoading(false)
      return
    }

    const loadData = async () => {
      setLoading(true)
      setError(null)
      try {
        const now = new Date()
        const from = new Date(now.getTime() - 2 * 60 * 60 * 1000) // 2 hours ago
        const to = new Date(now.getTime() + 4 * 60 * 60 * 1000) // 4 hours ahead

        const [chRes, epgRes] = await Promise.all([
          fetchChannels(libraryId),
          fetchEPG(libraryId, from.toISOString(), to.toISOString()),
        ])

        const ch = chRes.data.channels?.find((c) => c.id === channelId)
        if (!ch) {
          setError('Channel not found')
          return
        }
        setChannel(ch)

        // Filter programs for this channel
        const channelPrograms = epgRes.data.programs?.filter(
          (p) => p.channel_id === channelId
        ) ?? []
        setPrograms(channelPrograms)
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load channel data')
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [libraryId, channelId])

  const handleClose = useCallback(() => {
    navigate({ to: '/livetv', search: { libraryId } })
  }, [navigate, libraryId])

  const handleError = useCallback((err: string | null) => {
    if (err) {
      setError(err)
    }
  }, [])

  if (loading) {
    return (
      <div className="min-h-screen bg-neutral-950 text-white flex items-center justify-center">
        <div className="text-neutral-500">Loading stream...</div>
      </div>
    )
  }

  if (error && !channel) {
    return (
      <div className="min-h-screen bg-neutral-950 text-white flex flex-col items-center justify-center p-4">
        <p className="text-red-400 mb-4">{error}</p>
        <button
          onClick={handleClose}
          className="px-4 py-2 bg-neutral-800 hover:bg-neutral-700 text-white rounded-lg transition-colors"
        >
          Back to Live TV
        </button>
      </div>
    )
  }

  if (!channel) {
    return (
      <div className="min-h-screen bg-neutral-950 text-white flex items-center justify-center">
        <p className="text-neutral-500">Channel not found</p>
      </div>
    )
  }

  // Get current program for this channel
  const currentProgram = programs.find((p) => {
    const now = new Date()
    const start = new Date(p.start_time)
    const end = new Date(p.end_time)
    return now >= start && now <= end
  })

  return (
    <>
      {/* Main player */}
      <LiveStreamPlayer
        channel={channel}
        currentProgram={currentProgram}
        libraryId={libraryId ?? undefined}
        onClose={handleClose}
        onError={handleError}
        onDVRLimitReached={handleClose}
      />

      {/* Program info sidebar - shown when not in fullscreen */}
      <div className="fixed top-20 right-4 z-40 w-80 hidden xl:block">
        <EpgInfoBox programs={programs} showUpcoming={5} />
      </div>
    </>
  )
}

export const Route = createFileRoute('/_layout/livetv/play')({
  validateSearch: (search: Record<string, unknown>): LivetvPlaySearch => ({
    libraryId: Number(search.libraryId) || undefined,
    channelId: Number(search.channelId) || undefined,
  }),
  component: LivetvPlay,
})