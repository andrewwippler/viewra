import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router'
import { useState, useEffect, useCallback, useMemo } from 'react'
import { Play, Radio } from 'lucide-react'
import EpgGrid from '@/components/livetv/EpgGrid'
import EpgInfoBox from '@/components/livetv/EpgInfoBox'
import { fetchChannels, fetchEPG, type LiveChannel, type EpgProgram } from '@/lib/api/livetv'
import { useGetApiLibraries } from '@/lib/api'
import { extractLibraries } from '@/lib/utils/api'

interface LivetvSearch {
  libraryId?: number
  from?: string
  to?: string
  selectedChannelId?: number
}

const LivetvIndex = () => {
  const navigate = useNavigate()
  const search = useSearch({ from: Route.id })
  const libraryId = search.libraryId
  const selectedChannelId = search.selectedChannelId

  // Auto-detect first live_tv library when none is selected
  const { data: librariesData } = useGetApiLibraries()
  const allLibraries = extractLibraries(librariesData)

  useEffect(() => {
    if (!libraryId && allLibraries.length > 0) {
      const liveTv = allLibraries.find((l) => l.type === 'live_tv')
      if (liveTv?.id) {
        navigate({ to: '/livetv', search: { libraryId: liveTv.id }, replace: true })
      }
    }
  }, [libraryId, allLibraries, navigate])

  const [channels, setChannels] = useState<LiveChannel[]>([])
  const [programs, setPrograms] = useState<EpgProgram[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const timeWindow = useMemo(() => ({
    from: new Date(Date.now() - 2 * 60 * 60 * 1000),
    to: new Date(Date.now() + 48 * 60 * 60 * 1000),
  }), [])

  const loadData = useCallback(async (libId: number) => {
    setLoading(true)
    setError(null)
    try {
      const [chRes, epgRes] = await Promise.all([
        fetchChannels(libId),
        fetchEPG(libId, timeWindow.from.toISOString(), timeWindow.to.toISOString()),
      ])
      setChannels(chRes.data.channels ?? [])
      setPrograms(epgRes.data.programs ?? [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load Live TV data')
    } finally {
      setLoading(false)
    }
  }, [timeWindow])

  useEffect(() => {
    if (libraryId) {loadData(libraryId)}
    else {setLoading(false)}
  }, [libraryId, loadData])

  const handleTune = (channel: LiveChannel) => {
    navigate({
      to: '/livetv/play',
      search: { libraryId, channelId: channel.id },
    })
  }

  const handleChannelSelect = (channel: LiveChannel) => {
    navigate({
      to: '/livetv',
      search: { libraryId, selectedChannelId: channel.id },
    })
  }

  const handleProgramClick = (program: EpgProgram) => {
    navigate({
      to: '/livetv/event',
      search: { libraryId, programId: program.id },
    })
  }

  // Get selected channel and its programs for the EpgInfoBox
  const selectedChannel = useMemo(() => {
    return channels.find((ch) => ch.id === selectedChannelId) ?? null
  }, [channels, selectedChannelId])

  const selectedChannelPrograms = useMemo(() => {
    if (!selectedChannelId) {return []}
    return programs.filter((p) => p.channel_id === selectedChannelId)
  }, [programs, selectedChannelId])

  return (
    <div className="h-full flex flex-col bg-neutral-950">
      {/* Content */}
      {!libraryId ? (
        <div className="flex-1 flex items-center justify-center text-neutral-500">
          <div className="text-center">
            <Radio className="w-12 h-12 mx-auto mb-4 text-neutral-600" />
            <p className="text-lg mb-2">Select a Live TV library</p>
            <p className="text-sm">Go to Libraries and add a Live TV library to get started</p>
          </div>
        </div>
      ) : loading ? (
        <div className="flex-1 flex items-center justify-center">
          <div className="animate-pulse text-neutral-500">Loading channels...</div>
        </div>
      ) : error ? (
        <div className="flex-1 flex items-center justify-center text-red-400">
          <p>{error}</p>
        </div>
      ) : (
        <div className="flex-1 flex overflow-hidden">
          {/* EPG Grid */}
          <div className="flex-1 overflow-hidden">
            <EpgGrid
              channels={channels}
              programs={programs}
              timeWindow={timeWindow}
              onTune={handleTune}
              onChannelSelect={handleChannelSelect}
              onProgramClick={handleProgramClick}
            />
          </div>

          {/* Program Info Sidebar */}
          {selectedChannel && (
            <div className="w-80 shrink-0 border-l border-neutral-800 bg-neutral-900/50 overflow-y-auto p-4">
              {/* Selected channel header */}
              <div className="flex items-center gap-3 mb-4 pb-4 border-b border-neutral-800">
                {selectedChannel.logo_url && (
                  <img
                    src={selectedChannel.logo_url}
                    alt=""
                    className="w-12 h-12 object-contain rounded-lg bg-neutral-800 p-1"
                  />
                )}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-neutral-500 font-mono">
                      {selectedChannel.channel_number}
                    </span>
                    <span className="flex items-center gap-1 px-1.5 py-0.5 bg-red-600/30 text-red-400 text-[10px] font-bold rounded uppercase">
                      <span className="w-1.5 h-1.5 bg-red-400 rounded-full animate-pulse" />
                      LIVE
                    </span>
                  </div>
                  <h2 className="font-semibold text-white truncate">{selectedChannel.name}</h2>
                </div>
              </div>

              {/* Play button */}
              <button
                onClick={() => handleTune(selectedChannel)}
                className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-red-600 hover:bg-red-500 text-white font-semibold rounded-lg transition-colors mb-4 cursor-pointer"
              >
                <Play className="w-5 h-5" fill="currentColor" />
                Watch Now
              </button>

              {/* Current program */}
              {selectedChannel.current_program && (
                <div className="mb-4">
                  <h3 className="text-xs font-semibold text-neutral-500 uppercase tracking-wider mb-2">
                    Now Playing
                  </h3>
                  <div className="p-3 rounded-lg bg-neutral-800/50 border border-neutral-700">
                    <p className="font-medium text-white text-sm">
                      {selectedChannel.current_program.title}
                    </p>
                    {selectedChannel.current_program.description && (
                      <p className="text-xs text-neutral-400 mt-1 line-clamp-3">
                        {selectedChannel.current_program.description}
                      </p>
                    )}
                  </div>
                </div>
              )}

              {/* EPG Info Box */}
              <EpgInfoBox programs={selectedChannelPrograms} showUpcoming={5} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export const Route = createFileRoute('/_layout/livetv/')({
  validateSearch: (search: Record<string, unknown>): LivetvSearch => ({
    libraryId: Number(search.libraryId) || undefined,
    from: typeof search.from === 'string' ? search.from : undefined,
    to: typeof search.to === 'string' ? search.to : undefined,
    selectedChannelId: Number(search.selectedChannelId) || undefined,
  }),
  component: LivetvIndex,
})
