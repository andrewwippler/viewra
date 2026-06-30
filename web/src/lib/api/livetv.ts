import { customInstance } from './mutator'

export interface LiveChannel {
  id: number
  library_id: number
  channel_number: number
  name: string
  stream_url: string
  logo_url?: string
  group?: string
  epg_channel_id?: string
  enabled: boolean
  current_program?: EpgProgram
}

export interface EpgProgram {
  id: number
  channel_id: number
  start_time: string
  end_time: string
  title: string
  sub_title?: string
  description?: string
  category?: string
  episode_title?: string
  episode_num?: number
  season_num?: number
  is_new?: boolean
  is_movie?: boolean
}

interface ApiResponse<T> {
  data: T
  status: number
  headers: Headers
}

export const fetchChannels = (libraryId: number) =>
  customInstance<ApiResponse<{ channels: LiveChannel[] }>>({
    url: `/api/livetv/${libraryId}/channels`,
    method: 'GET',
  })

export const fetchEPG = (libraryId: number, from: string, to: string) =>
  customInstance<ApiResponse<{ programs: EpgProgram[] }>>({
    url: `/api/livetv/${libraryId}/epg?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    method: 'GET',
  })

export const fetchProgramById = (libraryId: number, programId: number) =>
  customInstance<ApiResponse<EpgProgram>>({
    url: `/api/livetv/${libraryId}/epg/programs/${programId}`,
    method: 'GET',
  })

export const scanChannels = (libraryId: number) =>
  customInstance<ApiResponse<{ message: string }>>({
    url: `/api/livetv/${libraryId}/scan-channels`,
    method: 'POST',
  })

export const scanEPG = (libraryId: number) =>
  customInstance<ApiResponse<{ message: string }>>({
    url: `/api/livetv/${libraryId}/scan-epg`,
    method: 'POST',
  })

export interface EPGChannelInfo {
  id: string
  name: string
}

export interface ChannelMapping {
  xmltv_channel_id: string
  channel_id?: number
  channel_name?: string
  epg_channel_name?: string
}

export interface MappingsResponse {
  epg_channels: EPGChannelInfo[]
  mappings: ChannelMapping[]
  channels: LiveChannel[]
  library_path: string
}

export const fetchMappings = (libraryId: number) =>
  customInstance<ApiResponse<MappingsResponse>>({
    url: `/api/livetv/${libraryId}/mappings`,
    method: 'GET',
  })

export const setMapping = (libraryId: number, xmltvChannelId: string, channelId: number) =>
  customInstance<ApiResponse<ChannelMapping>>({
    url: `/api/livetv/${libraryId}/mappings`,
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    data: { xmltv_channel_id: xmltvChannelId, channel_id: channelId },
  })

export const deleteMapping = (libraryId: number, xmltvChannelId: string) =>
  customInstance<ApiResponse<{ status: string }>>({
    url: `/api/livetv/${libraryId}/mappings/${encodeURIComponent(xmltvChannelId)}`,
    method: 'DELETE',
  })
