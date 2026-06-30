import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { nitpickyApi } from '../api/client'
import type { IdentifyRequest, RemoveMissingRequest } from '../api/client'

const QUERY_KEY = ['plugin', 'nitpicky-edits']

export const useNitpickyEditsAvailable = () => {
  return useQuery({
    queryKey: [...QUERY_KEY, 'available'],
    queryFn: async () => {
      try {
        const config = await nitpickyApi.getConfig()
        return { available: true, editModeRequired: config.edit_mode_required }
      } catch {
        return { available: false, editModeRequired: true }
      }
    },
    retry: false,
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  })
}

export const useIdentifyMovie = () => {
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: IdentifyRequest }) =>
      nitpickyApi.identifyMovie(id, data),
  })
}

export const useIdentifyTVShow = () => {
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: IdentifyRequest }) =>
      nitpickyApi.identifyTVShow(id, data),
  })
}

export const useDeleteEpisode = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => nitpickyApi.deleteEpisode(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tv-episodes'] })
    },
  })
}

export const useDeleteSeason = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => nitpickyApi.deleteSeason(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tv-episodes'] })
      queryClient.invalidateQueries({ queryKey: ['tv-show'] })
    },
  })
}

export const useDeleteShow = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => nitpickyApi.deleteShow(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tv-episodes'] })
      queryClient.invalidateQueries({ queryKey: ['tv-show'] })
    },
  })
}

export const useMarkWatchedMedia = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => nitpickyApi.markWatched(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] })
      queryClient.invalidateQueries({ queryKey: ['home'] })
    },
  })
}

export const useMarkUnwatchedMedia = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => nitpickyApi.markUnwatched(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] })
      queryClient.invalidateQueries({ queryKey: ['home'] })
    },
  })
}

export const useMarkSeasonWatched = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (seasonId: number) => nitpickyApi.markSeasonWatched(seasonId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] })
      queryClient.invalidateQueries({ queryKey: ['home'] })
    },
  })
}

export const useMarkSeasonUnwatched = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (seasonId: number) => nitpickyApi.markSeasonUnwatched(seasonId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['progress'] })
      queryClient.invalidateQueries({ queryKey: ['home'] })
    },
  })
}

export const useScanMissing = () => {
  return useQuery({
    queryKey: [...QUERY_KEY, 'scan-missing'],
    queryFn: () => nitpickyApi.scanMissing(),
    enabled: false,
    retry: false,
    staleTime: 0,
  })
}

export const useRemoveMissing = () => {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: RemoveMissingRequest) => nitpickyApi.removeMissing(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...QUERY_KEY, 'scan-missing'] })
      queryClient.invalidateQueries({ queryKey: ['tv-episodes'] })
      queryClient.invalidateQueries({ queryKey: ['tv-show'] })
      queryClient.invalidateQueries({ queryKey: ['movies'] })
    },
  })
}
