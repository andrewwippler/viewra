import { createFileRoute, useSearch, Link } from '@tanstack/react-router'
import { useState, useEffect, useCallback } from 'react'
import { ArrowLeft, Radio, Save, X } from 'lucide-react'
import { fetchChannels, fetchMappings, setMapping, deleteMapping, scanEPG, type LiveChannel, type MappingsResponse } from '@/lib/api/livetv'

interface MappingsSearch {
  libraryId?: number
}

const LivetvMappings = () => {
  const search = useSearch({ from: Route.id })
  const libraryId = search.libraryId

  const [data, setData] = useState<MappingsResponse | null>(null)
  const [channels, setChannels] = useState<LiveChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [savingId, setSavingId] = useState<string | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const loadData = useCallback(async (libId: number) => {
    setLoading(true)
    setError(null)
    try {
      const [mappingsRes, channelsRes] = await Promise.all([
        fetchMappings(libId),
        fetchChannels(libId),
      ])
      setData(mappingsRes.data)
      setChannels(channelsRes.data.channels ?? [])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load mappings')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (libraryId) {loadData(libraryId)}
    else {setLoading(false)}
  }, [libraryId, loadData])

  const handleSetMapping = async (xmltvChannelId: string, channelId: number) => {
    if (!libraryId) {return}
    setSavingId(xmltvChannelId)
    try {
      await setMapping(libraryId, xmltvChannelId, channelId)
      await loadData(libraryId)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to save mapping')
    } finally {
      setSavingId(null)
    }
  }

  const handleDeleteMapping = async (xmltvChannelId: string) => {
    if (!libraryId) {return}
    setDeleteId(xmltvChannelId)
    try {
      await deleteMapping(libraryId, xmltvChannelId)
      await loadData(libraryId)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to delete mapping')
    } finally {
      setDeleteId(null)
    }
  }

  const handleScanEPG = async () => {
    if (!libraryId) {return}
    setLoading(true)
    try {
      await scanEPG(libraryId)
      await loadData(libraryId)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'EPG scan failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="h-full flex flex-col bg-neutral-950">
      <div className="shrink-0 flex items-center justify-between px-4 py-3 border-b border-neutral-800 bg-neutral-900/50">
        <div className="flex items-center gap-3">
          <Link
            to="/livetv"
            search={{ libraryId }}
            className="p-1 rounded-lg hover:bg-neutral-800 text-neutral-400 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <Radio className="w-5 h-5 text-red-400" />
          <h1 className="text-lg font-semibold text-white">Channel Mapping</h1>
        </div>
        {libraryId && (
          <div className="text-sm text-neutral-500">
            <Link
              to="/livetv"
              search={{ libraryId }}
              className="text-blue-400 hover:text-blue-300 transition-colors"
            >
              &larr; Back to EPG Grid
            </Link>
          </div>
        )}
      </div>

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
          <div className="animate-pulse text-neutral-500">Loading mappings...</div>
        </div>
      ) : error ? (
        <div className="flex-1 flex items-center justify-center text-red-400">
          <p>{error}</p>
        </div>
      ) : (
        <div className="flex-1 overflow-auto p-4">
          <div className="max-w-3xl mx-auto">
            <p className="text-sm text-neutral-400 mb-4">
              Map XMLTV guide channels to M3U channels. Select an M3U channel for each guide channel
              to see its EPG data in the grid. Channels without a mapping will use their <code className="text-blue-400">tvg-id</code> attribute if it matches an XMLTV channel ID.
            </p>

            {data && data.mappings.length === 0 ? (
              <div className="text-center py-12 text-neutral-500">
                <p className="mb-2">No EPG channels found.</p>
                {data.library_path && (
                  <div className="mb-4 text-sm">
                    <p>Place <code className="text-blue-400">guide.xml</code> in the library folder:</p>
                    <p className="text-xs text-neutral-400 font-mono mt-1">{data.library_path}</p>
                  </div>
                )}
                <button
                  onClick={handleScanEPG}
                  disabled={loading}
                  className="flex items-center gap-2 mx-auto px-4 py-2 text-sm font-medium rounded-lg bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-colors"
                >
                  Scan for guide.xml
                </button>
              </div>
            ) : (
              <div className="space-y-2">
                {data && data.mappings.map((mapping) => {
                  return (
                    <div
                      key={mapping.xmltv_channel_id}
                      className="flex items-center gap-3 p-3 rounded-lg bg-neutral-900 border border-neutral-800"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-white truncate">
                            {mapping.epg_channel_name || mapping.xmltv_channel_id}
                          </span>
                          {!mapping.epg_channel_name && (
                            <span className="text-xs text-neutral-500 font-mono">
                              ({mapping.xmltv_channel_id})
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-neutral-500 mt-0.5">
                          XMLTV ID: <code className="text-neutral-400">{mapping.xmltv_channel_id}</code>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <select
                          className="bg-neutral-800 text-sm text-white border border-neutral-700 rounded-lg px-3 py-1.5 min-w-[180px] focus:outline-none focus:border-blue-500"
                          value={mapping.channel_id || ''}
                          onChange={(e) => {
                            const val = e.target.value
                            if (val) {
                              handleSetMapping(mapping.xmltv_channel_id, parseInt(val, 10))
                            }
                          }}
                          disabled={savingId === mapping.xmltv_channel_id}
                        >
                          <option value="">-- Unmapped --</option>
                          {channels.map(ch => (
                            <option key={ch.id} value={ch.id}>
                              {ch.name}{ch.epg_channel_id ? ` (tvg-id: ${ch.epg_channel_id})` : ''}
                            </option>
                          ))}
                        </select>
                        {mapping.channel_id ? (
                          <button
                            onClick={() => handleDeleteMapping(mapping.xmltv_channel_id)}
                            disabled={deleteId === mapping.xmltv_channel_id}
                            className="p-1.5 rounded-lg hover:bg-red-900/30 text-neutral-500 hover:text-red-400 transition-colors disabled:opacity-50"
                            title="Remove mapping"
                          >
                            <X className="w-4 h-4" />
                          </button>
                        ) : null}
                        {savingId === mapping.xmltv_channel_id && (
                          <Save className="w-4 h-4 text-blue-400 animate-pulse" />
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export const Route = createFileRoute('/_layout/livetv/mappings')({
  validateSearch: (search: Record<string, unknown>): MappingsSearch => ({
    libraryId: Number(search.libraryId) || undefined,
  }),
  component: LivetvMappings,
})
