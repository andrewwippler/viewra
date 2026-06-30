import { useState, useCallback } from 'react'
import { Link } from '@tanstack/react-router'
import { RefreshCw, Radio } from 'lucide-react'
import { scanChannels, scanEPG } from '@/lib/api/livetv'
import { useGetApiLibraries } from '@/lib/api'
import { extractLibraries } from '@/lib/utils/api'

interface LiveTvHeaderProps {
  libraryId?: number
  onScanComplete?: () => void
  onScanError?: (message: string) => void
}

export const LiveTvHeader = ({ libraryId: propLibraryId, onScanComplete, onScanError }: LiveTvHeaderProps) => {
  const { data: librariesData } = useGetApiLibraries()
  const allLibraries = extractLibraries(librariesData)
  const liveTvLib = allLibraries.find((l) => l.type === 'live_tv')
  const libraryId = propLibraryId ?? liveTvLib?.id

  const [scanning, setScanning] = useState(false)

  const handleScanChannels = useCallback(async () => {
    if (!libraryId) {
      return
    }
    setScanning(true)
    try {
      await scanChannels(libraryId)
      onScanComplete?.()
    } catch {
      onScanError?.('Scan failed')
    } finally {
      setScanning(false)
    }
  }, [libraryId, onScanComplete, onScanError])

  const handleScanEPG = useCallback(async () => {
    if (!libraryId) {
      return
    }
    setScanning(true)
    try {
      await scanEPG(libraryId)
      onScanComplete?.()
    } catch {
      onScanError?.('EPG scan failed')
    } finally {
      setScanning(false)
    }
  }, [libraryId, onScanComplete, onScanError])

  return (
    <div className="shrink-0 flex items-center justify-between px-4 py-3 border-b border-neutral-800 bg-neutral-900/50">
      <div className="flex items-center gap-3">
        <Radio className="w-5 h-5 text-red-400" />
        <h1 className="text-lg font-semibold text-white">Live TV</h1>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={handleScanChannels}
          disabled={scanning || !libraryId}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 disabled:opacity-50 transition-colors cursor-pointer"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
          Scan Channels
        </button>
        <button
          onClick={handleScanEPG}
          disabled={scanning || !libraryId}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 disabled:opacity-50 transition-colors cursor-pointer"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${scanning ? 'animate-spin' : ''}`} />
          Scan EPG
        </button>
        <Link
          to="/livetv/mappings"
          search={{ libraryId }}
          disabled={!libraryId}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 disabled:opacity-50 transition-colors"
        >
          Mappings
        </Link>
      </div>
    </div>
  )
}
