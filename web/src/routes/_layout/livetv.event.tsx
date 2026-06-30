import { createFileRoute, Link, useSearch } from '@tanstack/react-router'
import { useState, useEffect, useMemo } from 'react'
import { Clock, Calendar, Film, Tv, Music, BookOpen, Gamepad2, Newspaper, ArrowLeft, Play, Radio } from 'lucide-react'
import { fetchProgramById, fetchChannels, fetchEPG, type EpgProgram, type LiveChannel } from '@/lib/api/livetv'
import { useGetApiLibraries } from '@/lib/api'
import { extractLibraries } from '@/lib/utils/api'

interface EventSearch {
  libraryId?: number
  programId?: number
}

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  movie: <Film className="w-4 h-4" />,
  film: <Film className="w-4 h-4" />,
  series: <Tv className="w-4 h-4" />,
  tv: <Tv className="w-4 h-4" />,
  news: <Newspaper className="w-4 h-4" />,
  sports: <Gamepad2 className="w-4 h-4" />,
  music: <Music className="w-4 h-4" />,
  documentary: <BookOpen className="w-4 h-4" />,
  kids: <Tv className="w-4 h-4" />,
  entertainment: <Tv className="w-4 h-4" />,
}

const CATEGORY_COLORS: Record<string, string> = {
  movie: 'bg-purple-500/20 border-purple-500/40 text-purple-300',
  film: 'bg-purple-500/20 border-purple-500/40 text-purple-300',
  series: 'bg-blue-500/20 border-blue-500/40 text-blue-300',
  tv: 'bg-blue-500/20 border-blue-500/40 text-blue-300',
  news: 'bg-cyan-500/20 border-cyan-500/40 text-cyan-300',
  sports: 'bg-green-500/20 border-green-500/40 text-green-300',
  music: 'bg-orange-500/20 border-orange-500/40 text-orange-300',
  documentary: 'bg-teal-500/20 border-teal-500/40 text-teal-300',
  kids: 'bg-pink-500/20 border-pink-500/40 text-pink-300',
  entertainment: 'bg-amber-500/20 border-amber-500/40 text-amber-300',
  default: 'bg-neutral-500/20 border-neutral-500/40 text-neutral-300',
}

const getCategoryStyle = (category?: string) => {
  if (!category) {return CATEGORY_COLORS.default}
  const key = category.toLowerCase()
  for (const [k, v] of Object.entries(CATEGORY_COLORS)) {
    if (key.includes(k)) {return v}
  }
  return CATEGORY_COLORS.default
}

const getCategoryIcon = (category?: string) => {
  if (!category) {return <Clock className="w-4 h-4" />}
  const key = category.toLowerCase()
  for (const [k, v] of Object.entries(CATEGORY_ICONS)) {
    if (key.includes(k)) {return v}
  }
  return <Clock className="w-4 h-4" />
}

const formatTime = (dateStr: string) => {
  return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const today = new Date()
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)

  if (date.toDateString() === today.toDateString()) {return 'Today'}
  if (date.toDateString() === tomorrow.toDateString()) {return 'Tomorrow'}
  return date.toLocaleDateString([], { weekday: 'short', month: 'short', day: 'numeric' })
}

const formatDuration = (startStr: string, endStr: string) => {
  const start = new Date(startStr)
  const end = new Date(endStr)
  const minutes = Math.round((end.getTime() - start.getTime()) / 60000)
  if (minutes < 60) {return `${minutes}m`}
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
}

const LivetvEvent = () => {
  const search = useSearch({ from: Route.id })
  const { libraryId, programId } = search

  const [program, setProgram] = useState<EpgProgram | null>(null)
  const [channel, setChannel] = useState<LiveChannel | null>(null)
  const [channelPrograms, setChannelPrograms] = useState<EpgProgram[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const { data: librariesData } = useGetApiLibraries()
  const allLibraries = extractLibraries(librariesData)
  const resolvedLibId = libraryId ?? allLibraries.find((l) => l.type === 'live_tv')?.id

  useEffect(() => {
    if (!resolvedLibId || !programId) {return}

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const [progRes, chRes] = await Promise.all([
          fetchProgramById(resolvedLibId, programId),
          fetchChannels(resolvedLibId),
        ])
        const prog = progRes.data
        setProgram(prog)

        const ch = chRes.data.channels.find((c) => c.id === prog.channel_id) ?? null
        setChannel(ch)

        if (ch) {
          const from = new Date(new Date(prog.start_time).getTime() - 6 * 60 * 60 * 1000).toISOString()
          const to = new Date(new Date(prog.end_time).getTime() + 6 * 60 * 60 * 1000).toISOString()
          const epgRes = await fetchEPG(resolvedLibId, from, to)
          const channelProgs = epgRes.data.programs.filter((p) => p.channel_id === ch.id)
          setChannelPrograms(channelProgs)
        }
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load program details')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [resolvedLibId, programId])

  const isLive = useMemo(() => {
    if (!program) {return false}
    const now = new Date()
    return now >= new Date(program.start_time) && now <= new Date(program.end_time)
  }, [program])

  const nearbyPrograms = useMemo(() => {
    if (!program || channelPrograms.length === 0) {return []}
    return channelPrograms.filter((p) => p.id !== program.id)
  }, [program, channelPrograms])

  if (!resolvedLibId || !programId) {
    return (
      <div className="h-full flex items-center justify-center bg-neutral-950 text-neutral-500">
        <p>No program selected</p>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center bg-neutral-950">
        <div className="animate-pulse text-neutral-500">Loading program details...</div>
      </div>
    )
  }

  if (error || !program) {
    return (
      <div className="h-full flex flex-col items-center justify-center bg-neutral-950 text-red-400 gap-4">
        <p>{error || 'Program not found'}</p>
        <Link to="/livetv" search={{ libraryId: resolvedLibId }} className="text-sm text-neutral-400 hover:text-white underline">
          Back to Live TV
        </Link>
      </div>
    )
  }

  const catStyle = getCategoryStyle(program.category)
  const icon = getCategoryIcon(program.category)

  return (
    <div className="h-full flex flex-col bg-neutral-950 text-white overflow-y-auto">
      {/* Header */}
      <div className="shrink-0 flex items-center gap-3 px-4 py-3 border-b border-neutral-800 bg-neutral-900/50">
        <Link
          to="/livetv"
          search={{ libraryId: resolvedLibId, selectedChannelId: program.channel_id }}
          className="flex items-center gap-1.5 text-sm text-neutral-400 hover:text-white transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Live TV
        </Link>
        <div className="flex items-center gap-2 ml-auto">
          <Radio className="w-5 h-5 text-red-400" />
          <span className="text-sm font-semibold">Program Details</span>
        </div>
      </div>

      <div className="flex-1 max-w-4xl mx-auto w-full p-6 space-y-6">
        {/* Program card */}
        <div className={`rounded-xl border p-6 ${catStyle}`}>
          {/* Channel info */}
          {channel && (
            <div className="flex items-center gap-3 mb-4 pb-4 border-b border-white/10">
              {channel.logo_url && (
                <img src={channel.logo_url} alt="" className="w-10 h-10 object-contain rounded-lg bg-neutral-800 p-1" />
              )}
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-neutral-500 font-mono">{channel.channel_number}</span>
                  {isLive && (
                    <span className="flex items-center gap-1 px-1.5 py-0.5 bg-red-600/30 text-red-400 text-[10px] font-bold rounded uppercase">
                      <span className="w-1.5 h-1.5 bg-red-400 rounded-full animate-pulse" />
                      LIVE
                    </span>
                  )}
                </div>
                <p className="font-medium text-white">{channel.name}</p>
              </div>
              <Link
                to="/livetv/play"
                search={{ libraryId: resolvedLibId, channelId: channel.id }}
                className="ml-auto flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-500 text-white font-semibold rounded-lg transition-colors text-sm"
              >
                <Play className="w-4 h-4" fill="currentColor" />
                Watch Now
              </Link>
            </div>
          )}

          {/* Program header */}
          <div className="flex items-start gap-3">
            <span className="mt-1 shrink-0 opacity-70">{icon}</span>
            <div className="min-w-0 flex-1">
              <h1 className="text-2xl font-bold text-white">{program.title}</h1>
              {program.sub_title && (
                <p className="text-base text-neutral-300 mt-1">{program.sub_title}</p>
              )}
            </div>
          </div>

          {/* Time info */}
          <div className="flex flex-wrap items-center gap-4 mt-4 text-sm text-neutral-400">
            <span className="flex items-center gap-1.5">
              <Calendar className="w-4 h-4" />
              {formatDate(program.start_time)}
            </span>
            <span className="flex items-center gap-1.5">
              <Clock className="w-4 h-4" />
              {formatTime(program.start_time)} - {formatTime(program.end_time)}
            </span>
            <span className="text-neutral-500">
              {formatDuration(program.start_time, program.end_time)}
            </span>
          </div>

          {/* Episode info */}
          {(program.season_num !== undefined || program.episode_num !== undefined) && (
            <div className="mt-4 text-sm text-neutral-300">
              Season {program.season_num}
              {program.episode_num !== undefined && <> &middot; Episode {program.episode_num}</>}
              {program.episode_title && <> &middot; {program.episode_title}</>}
            </div>
          )}

          {/* Badges */}
          <div className="mt-4 flex flex-wrap gap-2">
            {isLive && (
              <span className="px-2 py-0.5 bg-red-600/30 text-red-300 text-xs font-bold rounded uppercase">
                Live
              </span>
            )}
            {program.is_new && (
              <span className="px-2 py-0.5 bg-green-600/30 text-green-300 text-xs font-medium rounded uppercase">
                New
              </span>
            )}
            {program.is_movie && (
              <span className="px-2 py-0.5 bg-purple-600/30 text-purple-300 text-xs font-medium rounded uppercase">
                Movie
              </span>
            )}
            {program.category && (
              <span className="px-2 py-0.5 bg-white/10 text-neutral-300 text-xs font-medium rounded uppercase">
                {program.category}
              </span>
            )}
          </div>

          {/* Description */}
          {program.description && (
            <div className="mt-6">
              <h3 className="text-sm font-semibold text-neutral-500 uppercase tracking-wider mb-2">Description</h3>
              <p className="text-sm text-neutral-300 leading-relaxed whitespace-pre-wrap">{program.description}</p>
            </div>
          )}
        </div>

        {/* Other programs on this channel */}
        {nearbyPrograms.length > 0 && (
          <div>
            <h2 className="text-lg font-semibold text-white mb-4">
              {channel ? `More on ${channel.name}` : 'More on this channel'}
            </h2>
            <div className="space-y-2">
              {nearbyPrograms.map((p) => {
                const isActive = p.id === program.id
                const pCatStyle = getCategoryStyle(p.category)
                const pIcon = getCategoryIcon(p.category)
                return (
                  <Link
                    key={p.id}
                    to="/livetv/event"
                    search={{ libraryId: resolvedLibId, programId: p.id }}
                    className={`flex items-center gap-3 p-3 rounded-lg border transition-colors ${
                      isActive
                        ? 'border-red-500/40 bg-red-500/10'
                        : 'border-neutral-800 bg-neutral-900/50 hover:bg-neutral-800/50 hover:border-neutral-700'
                    }`}
                  >
                    <span className="shrink-0 opacity-60">{pIcon}</span>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium truncate">{p.title}</p>
                      <p className="text-xs text-neutral-500">
                        {formatTime(p.start_time)} - {formatTime(p.end_time)} &middot; {formatDuration(p.start_time, p.end_time)}
                      </p>
                    </div>
                    <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${pCatStyle}`}>
                      {p.category || 'General'}
                    </span>
                  </Link>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_layout/livetv/event')({
  validateSearch: (search: Record<string, unknown>): EventSearch => ({
    libraryId: Number(search.libraryId) || undefined,
    programId: Number(search.programId) || undefined,
  }),
  component: LivetvEvent,
})
