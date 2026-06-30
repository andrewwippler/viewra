/**
 * Media API Client
 * Wrapper around generated API functions
 */

import { deleteApiMediaId } from './generated/media/media'

export const mediaApi = {
  /**
   * Delete a media item (database only, does not delete file on disk)
   */
  deleteMedia: (id: number) => deleteApiMediaId(id),
}