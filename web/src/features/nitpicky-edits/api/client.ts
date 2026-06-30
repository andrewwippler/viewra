import { pluginApi } from '@/lib/api/pluginApi'

const PLUGIN_ID = 'nitpicky-edits'

export interface PluginConfig {
  edit_mode_required: boolean
}

export interface IdentifyRequest {
  imdb_id: string
}

export interface IdentifyResponse {
  status: string
  movie_id?: number
  show_id?: number
}

export interface MarkProgressResponse {
  status: string
  media_id: number
  is_watched: boolean
}

export interface MarkSeasonProgressResponse {
  status: string
  season_id: number
  is_watched: boolean
}

export interface MissingItem {
  media_id: number
  show_id?: number
  item_type: 'movie' | 'tv_episode' | 'tv_show'
  title: string
  file_path: string
  year?: number
  show_title?: string
  season_number?: number
  episode_number?: number
  library_name?: string
}

export interface RemoveMissingRequest {
  media_ids: number[]
  show_ids: number[]
}

export const nitpickyApi = {
  getConfig: () =>
    pluginApi.get<PluginConfig>(PLUGIN_ID, '/config'),

  identifyMovie: (id: number, data: IdentifyRequest) =>
    pluginApi.post<IdentifyResponse>(PLUGIN_ID, `/identify/movie/${id}`, data),

  identifyTVShow: (id: number, data: IdentifyRequest) =>
    pluginApi.post<IdentifyResponse>(PLUGIN_ID, `/identify/tv/${id}`, data),

  deleteEpisode: (id: number) =>
    pluginApi.delete(PLUGIN_ID, `/episode/${id}`),

  deleteSeason: (id: number) =>
    pluginApi.delete(PLUGIN_ID, `/season/${id}`),

  deleteShow: (id: number) =>
    pluginApi.delete(PLUGIN_ID, `/show/${id}`),

  markWatched: (id: number) =>
    pluginApi.post<MarkProgressResponse>(PLUGIN_ID, `/progress/${id}/watched`),

  markUnwatched: (id: number) =>
    pluginApi.post<MarkProgressResponse>(PLUGIN_ID, `/progress/${id}/unwatched`),

  markSeasonWatched: (seasonId: number) =>
    pluginApi.post<MarkSeasonProgressResponse>(PLUGIN_ID, `/progress/season/${seasonId}/watched`),

  markSeasonUnwatched: (seasonId: number) =>
    pluginApi.post<MarkSeasonProgressResponse>(PLUGIN_ID, `/progress/season/${seasonId}/unwatched`),

  scanMissing: () =>
    pluginApi.get<MissingItem[]>(PLUGIN_ID, '/scan-missing'),

  removeMissing: (data: RemoveMissingRequest) =>
    pluginApi.post<{ status: string }>(PLUGIN_ID, '/remove-missing', data),
}
