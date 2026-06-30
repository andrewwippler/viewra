/**
 * EpgInfoBox Component
 *
 * Displays program guide information in a standardized format.
 * Shows current and upcoming programs for a channel.
 */

import { useMemo } from 'react'
import { Clock, Calendar, Film, Tv, Music, BookOpen, Gamepad2, Newspaper } from 'lucide-react'
import type { EpgProgram } from '@/lib/api/livetv'

interface EpgInfoBoxProps {
  programs: EpgProgram[]
  showUpcoming?: number
  className?: string
}

// Map categories to icons
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

// Map categories to colors
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
  const date = new Date(dateStr)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatDuration = (startStr: string, endStr: string) => {
  const start = new Date(startStr)
  const end = new Date(endStr)
  const minutes = Math.round((end.getTime() - start.getTime()) / 60000)
  if (minutes < 60) {
    return `${minutes}m`
  }
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
}

const ProgramCard = ({ program }: { program: EpgProgram }) => {
  const isLive = useMemo(() => {
    const now = new Date()
    const start = new Date(program.start_time)
    const end = new Date(program.end_time)
    return now >= start && now <= end
  }, [program.start_time, program.end_time])

  const catStyle = getCategoryStyle(program.category)
  const icon = getCategoryIcon(program.category)

  return (
    <div
      className={`relative p-3 rounded-lg border ${catStyle} ${
        isLive ? 'ring-2 ring-red-500/50' : ''
      }`}
    >
      {/* Live indicator */}
      {isLive && (
        <div className="absolute -top-1 -right-1 flex items-center gap-1 px-1.5 py-0.5 bg-red-600 text-white text-[10px] font-bold rounded uppercase">
          <span className="w-1.5 h-1.5 bg-white rounded-full animate-pulse" />
          LIVE
        </div>
      )}

      {/* Header */}
      <div className="flex items-start gap-2 mb-2">
        <span className="mt-0.5 shrink-0 opacity-70">{icon}</span>
        <div className="min-w-0 flex-1">
          <h4 className="font-medium text-sm leading-tight truncate">{program.title}</h4>
          {program.sub_title && (
            <p className="text-xs opacity-70 truncate mt-0.5">{program.sub_title}</p>
          )}
        </div>
      </div>

      {/* Time info */}
      <div className="flex items-center gap-3 text-xs opacity-80">
        <span className="flex items-center gap-1">
          <Clock className="w-3 h-3" />
          {formatTime(program.start_time)} - {formatTime(program.end_time)}
        </span>
        <span className="flex items-center gap-1">
          <Calendar className="w-3 h-3" />
          {formatDuration(program.start_time, program.end_time)}
        </span>
      </div>

      {/* Episode info */}
      {(program.season_num !== undefined || program.episode_num !== undefined) && (
        <div className="mt-2 text-xs opacity-80">
          {program.season_num !== undefined && `S${program.season_num}`}
          {program.season_num !== undefined && program.episode_num !== undefined && ':'}
          {program.episode_num !== undefined && `E${program.episode_num}`}
          {program.episode_title && ` - ${program.episode_title}`}
        </div>
      )}

      {/* Description */}
      {program.description && (
        <p className="mt-2 text-xs opacity-70 line-clamp-3 leading-relaxed">
          {program.description}
        </p>
      )}

      {/* Badges */}
      <div className="mt-2 flex flex-wrap gap-1">
        {program.is_new && (
          <span className="px-1.5 py-0.5 bg-green-600/30 text-green-300 text-[10px] font-medium rounded uppercase">
            New
          </span>
        )}
        {program.is_movie && (
          <span className="px-1.5 py-0.5 bg-purple-600/30 text-purple-300 text-[10px] font-medium rounded uppercase">
            Movie
          </span>
        )}
        {program.category && (
          <span className="px-1.5 py-0.5 bg-white/10 text-white/80 text-[10px] font-medium rounded uppercase">
            {program.category}
          </span>
        )}
      </div>
    </div>
  )
}

export const EpgInfoBox = ({
  programs,
  showUpcoming = 3,
  className = '',
}: EpgInfoBoxProps) => {
  // Filter to current and upcoming programs
  const displayPrograms = useMemo(() => {
    const now = new Date()
    const sorted = [...programs]
      .filter((p) => new Date(p.end_time) > now)
      .sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())
      .slice(0, showUpcoming)
    return sorted
  }, [programs, showUpcoming])

  if (displayPrograms.length === 0) {
    return (
      <div className={`p-4 bg-neutral-900/50 rounded-lg border border-neutral-800 ${className}`}>
        <p className="text-sm text-neutral-500 text-center">No program information available</p>
      </div>
    )
  }

  const currentProgram = displayPrograms[0]
  const now = new Date()
  const isNow = now >= new Date(currentProgram.start_time) && now <= new Date(currentProgram.end_time)

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Current program highlight */}
      {isNow && (
        <div>
          <h3 className="text-xs font-semibold text-neutral-500 uppercase tracking-wider mb-2">
            Now Playing
          </h3>
          <ProgramCard program={currentProgram} />
        </div>
      )}

      {/* Upcoming programs */}
      {displayPrograms.length > 1 && (
        <div>
          <h3 className="text-xs font-semibold text-neutral-500 uppercase tracking-wider mb-2">
            {isNow ? 'Coming Up' : 'Schedule'}
          </h3>
          <div className="space-y-2">
            {displayPrograms.slice(isNow ? 1 : 0).map((program) => (
              <ProgramCard key={program.id} program={program} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default EpgInfoBox