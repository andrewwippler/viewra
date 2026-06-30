import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { fetchChannels, type LiveChannel } from '@/lib/api/livetv'
import { useWebOSInputNavigation } from '@/lib/hooks'
import { useGetApiLibraries } from '@/lib/api'
import { extractLibraries } from '@/lib/utils/api'

interface WebosLivetvSearch {
  libraryId?: number
}

const WebosLivetvIndex = () => {
  const navigate = useNavigate()
  const search = useSearch({ from: Route.id })
  const libraryId = search.libraryId

  const [channels, setChannels] = useState<LiveChannel[]>([])
  const [loading, setLoading] = useState(true)

  useWebOSInputNavigation({
    enabled: true,
    rowSelector: '[data-row]',
    onBackPress: () => navigate({ to: '/webos' } as never),
  })

  const { data: librariesData } = useGetApiLibraries()
  const allLibraries = extractLibraries(librariesData)

  useEffect(() => {
    if (!libraryId && allLibraries.length > 0) {
      const liveTv = allLibraries.find((l) => l.type === 'live_tv')
      if (liveTv?.id) {
        navigate({ to: '/webos/livetv', search: { libraryId: liveTv.id }, replace: true })
      }
    }
  }, [libraryId, allLibraries, navigate])

  useEffect(() => {
    if (!libraryId) {
      setLoading(false)
      return
    }
    setLoading(true)
    fetchChannels(libraryId)
      .then((res) => {
        setChannels(res.data.channels ?? [])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [libraryId])

  const handleTune = (ch: LiveChannel) => {
    navigate({
      to: '/webos/livetv/play',
      search: { libraryId, channelId: ch.id },
    })
  }

  if (!libraryId) {
    return (
      <div className="min-h-screen bg-neutral-950 text-white flex items-center justify-center p-4">
        <p className="text-lg text-neutral-500 text-center">No Live TV library selected</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-neutral-950 text-white flex items-center justify-center">
        <p className="text-lg text-neutral-500">Loading channels...</p>
      </div>
    )
  }

  const formatTime = (iso: string) => {
    const d = new Date(iso)
    const h = d.getHours()
    const m = d.getMinutes()
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`
  }

  return (
    <div className="min-h-screen bg-neutral-950 text-white p-4">
      <div data-row className="space-y-2">
        {channels.map((ch) => (
          <button
            key={ch.id}
            type="button"
            className="w-full flex items-center gap-4 p-3 rounded bg-neutral-800/60 hover:bg-neutral-700/60 transition-colors text-left"
            onClick={() => handleTune(ch)}
          >
            <span className="text-lg text-neutral-500 font-mono w-16 shrink-0 text-center">
              {ch.channel_number}
            </span>
            <div className="min-w-0 flex-1 flex items-center gap-3 overflow-hidden">
              <span className="text-base font-medium truncate text-white/90">{ch.name}</span>
              {ch.current_program && (
                <span className="text-base text-neutral-400 truncate">
                  {formatTime(ch.current_program.start_time)}-{formatTime(ch.current_program.end_time)} {ch.current_program.title}
                </span>
              )}
            </div>
            {ch.logo_url && (
              <img src={ch.logo_url} alt="" className="w-10 h-10 object-contain rounded shrink-0" />
            )}
          </button>
        ))}
      </div>

      {channels.length === 0 && (
        <div className="text-center py-16">
          <p className="text-base text-neutral-500">No channels found</p>
          <p className="text-sm text-neutral-600 mt-2">Scan channels from the web interface</p>
        </div>
      )}
    </div>
  )
}

export const Route = createFileRoute('/webos/livetv/')({
  validateSearch: (search: Record<string, unknown>): WebosLivetvSearch => ({
    libraryId: Number(search.libraryId) || undefined,
  }),
  component: WebosLivetvIndex,
})
