import { useMemo, useRef, useState } from 'react'
import type { LiveChannel, EpgProgram } from '@/lib/api/livetv'

interface EpgGridProps {
  channels: LiveChannel[]
  programs: EpgProgram[]
  timeWindow: { from: Date; to: Date }
  onTune: (channel: LiveChannel) => void
  onChannelSelect?: (channel: LiveChannel) => void
  onProgramClick?: (program: EpgProgram) => void
}

const PIXELS_PER_HOUR = 180
const CHANNEL_WIDTH = 220
const HEADER_HEIGHT = 48
const ROW_HEIGHT = 64
const CATEGORY_COLORS: Record<string, string> = {
  news: 'bg-blue-500/30 border-blue-500/50',
  sports: 'bg-green-500/30 border-green-500/50',
  movie: 'bg-purple-500/30 border-purple-500/50',
  film: 'bg-purple-500/30 border-purple-500/50',
  entertainment: 'bg-amber-500/30 border-amber-500/50',
  documentary: 'bg-cyan-500/30 border-cyan-500/50',
  kids: 'bg-pink-500/30 border-pink-500/50',
  music: 'bg-orange-500/30 border-orange-500/50',
  default: 'bg-neutral-600/30 border-neutral-600/50',
}

const getCategoryStyle = (category?: string) => {
  if (!category) {return CATEGORY_COLORS.default}
  const key = category.toLowerCase()
  for (const [k, v] of Object.entries(CATEGORY_COLORS)) {
    if (key.includes(k)) {return v}
  }
  return CATEGORY_COLORS.default
}

const formatTime = (d: Date) => {
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatHour = (d: Date) => {
  return d.toLocaleTimeString([], { hour: 'numeric' })
}

const msToPixels = (ms: number) => {
  return (ms / (1000 * 60 * 60)) * PIXELS_PER_HOUR
}

const dateToOffset = (d: Date, from: Date) => {
  return msToPixels(d.getTime() - from.getTime())
}

const EpgGrid = ({ channels, programs, timeWindow, onTune, onChannelSelect, onProgramClick }: EpgGridProps) => {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [selectedChannelId, setSelectedChannelId] = useState<number | null>(null)

  const totalWidth = msToPixels(timeWindow.to.getTime() - timeWindow.from.getTime())

  const hours = useMemo(() => {
    const h: Date[] = []
    const cursor = new Date(timeWindow.from)
    cursor.setMinutes(0, 0, 0)
    while (cursor <= timeWindow.to) {
      h.push(new Date(cursor))
      cursor.setHours(cursor.getHours() + 1)
    }
    return h
  }, [timeWindow])

  const nowPosition = useMemo(() => {
    const now = new Date()
    if (now >= timeWindow.from && now <= timeWindow.to) {
      return dateToOffset(now, timeWindow.from)
    }
    return -1
  }, [timeWindow])

  const programMap = useMemo(() => {
    const map = new Map<number, EpgProgram[]>()
    for (const p of programs) {
      const existing = map.get(p.channel_id) || []
      existing.push(p)
      map.set(p.channel_id, existing)
    }
    return map
  }, [programs])



  return (
    <div className="flex flex-col h-full bg-neutral-950 text-white">
      {/* Time header */}
      <div className="flex shrink-0" style={{ height: HEADER_HEIGHT }}>
        <div
          className="shrink-0 border-r border-neutral-800 bg-neutral-900 flex items-end px-3 pb-2"
          style={{ width: CHANNEL_WIDTH }}
        >
          <span className="text-xs text-neutral-500 font-medium uppercase tracking-wider">Channels</span>
        </div>
        <div
          className="overflow-hidden"
          ref={scrollRef}
        >
          <div className="relative h-full" style={{ width: totalWidth }}>
            {hours.map((h, i) => (
              <div
                key={i}
                className="absolute top-0 h-full border-l border-neutral-800/50"
                style={{ left: dateToOffset(h, timeWindow.from) }}
              >
                <span className="text-xs text-neutral-500 px-2 py-1 inline-block">
                  {formatHour(h)}
                </span>
              </div>
            ))}
            {nowPosition >= 0 && (
              <div
                className="absolute top-0 w-0.5 bg-red-500 shadow-lg shadow-red-500/50 z-10"
                style={{ left: nowPosition, height: HEADER_HEIGHT }}
              />
            )}
          </div>
        </div>
      </div>

      {/* Channel rows */}
      <div className="flex flex-1 overflow-hidden">
        {/* Channel sidebar (sticky) */}
        <div className="shrink-0 border-r border-neutral-800 bg-neutral-900/50 overflow-y-auto" style={{ width: CHANNEL_WIDTH }}>
          {channels.map((ch) => (
            <div
              key={ch.id}
              className={`flex items-center gap-2 px-3 border-b border-neutral-800/50 cursor-pointer transition-colors hover:bg-neutral-800/50 ${
                selectedChannelId === ch.id ? 'bg-neutral-800' : ''
              }`}
              style={{ height: ROW_HEIGHT }}
              onClick={() => {
                setSelectedChannelId(ch.id)
                onChannelSelect?.(ch)
              }}
              onDoubleClick={() => {
                onTune(ch)
              }}
            >
              <span className="text-xs text-neutral-500 font-mono w-8 shrink-0">
                {ch.channel_number}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium truncate">{ch.name}</p>
                {ch.current_program && (
                  <p className="text-xs text-neutral-400 truncate">
                    {ch.current_program.title}
                  </p>
                )}
              </div>
              {ch.logo_url && (
                <img src={ch.logo_url} alt="" className="w-6 h-6 object-contain shrink-0 rounded" />
              )}
            </div>
          ))}
        </div>

        {/* Program grid (scrollable) */}
        <div className="flex-1 overflow-auto" ref={scrollRef}>
          <div className="relative" style={{ width: totalWidth }}>
            {/* Hour grid lines */}
            {hours.map((h, i) => (
              <div
                key={i}
                className="absolute top-0 bottom-0 border-l border-neutral-800/30 pointer-events-none"
                style={{ left: dateToOffset(h, timeWindow.from) }}
              />
            ))}

            {/* Current time indicator (vertical line) */}
            {nowPosition >= 0 && (
              <div
                className="absolute top-0 bottom-0 w-0.5 bg-red-500 shadow-lg shadow-red-500/50 z-10 pointer-events-none"
                style={{ left: nowPosition }}
              />
            )}

            {/* Program blocks */}
            {channels.map((ch, rowIndex) => {
      let channelPrograms = programMap.get(ch.id) || []
      if (channelPrograms.length === 0 && ch.current_program) {
        channelPrograms = [ch.current_program]
      }
              const top = rowIndex * ROW_HEIGHT
              return (
                <div key={ch.id} className="absolute left-0 right-0" style={{ top, height: ROW_HEIGHT }}>
                  {channelPrograms.map((p) => {
                    const left = dateToOffset(new Date(p.start_time), timeWindow.from)
                    const width = Math.max(
                      msToPixels(
                        new Date(p.end_time).getTime() - new Date(p.start_time).getTime()
                      ),
                      2
                    )
                    const catStyle = getCategoryStyle(p.category)
                    return (
                      <button
                        key={p.id}
                        className={`absolute inset-y-0.5 rounded border-l-2 overflow-hidden text-left transition-all hover:brightness-110 hover:scale-y-[1.02] ${catStyle}`}
                        style={{ left, width }}
                        onClick={() => onProgramClick?.(p)}
                        title={p.description || p.title}
                      >
                        <div className="px-1 py-0.5 h-full flex flex-col justify-center">
                          <p className="text-xs font-medium leading-tight truncate">{p.title}</p>
                          <p className="text-[10px] text-white/60 leading-tight truncate">
                            {formatTime(new Date(p.start_time))}
                          </p>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}

export default EpgGrid
