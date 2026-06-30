import { useState } from 'react'
import { Modal, ModalContent, ModalFooter, Button, Input } from '@/components/ui'

interface IdentifyModalProps {
  isOpen: boolean
  onClose: () => void
  mediaType: 'movie' | 'tv'
  mediaTitle: string
  onIdentify: (data: { imdb_id: string }) => void
  isPending?: boolean
}

export const IdentifyModal = ({ isOpen, onClose, mediaType, mediaTitle, onIdentify, isPending }: IdentifyModalProps) => {
  const [imdbId, setImdbId] = useState('')

  const handleSubmit = () => {
    onIdentify({ imdb_id: imdbId })
  }

  const handleClose = () => {
    setImdbId('')
    onClose()
  }

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title={`Identify ${mediaType === 'movie' ? 'Movie' : 'TV Show'}`} size="sm">
      <ModalContent>
        <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-4">
          Set IMDb ID for <strong>{mediaTitle}</strong>
        </p>
        <div>
          <label className="block text-sm font-medium mb-1">IMDb ID</label>
          <Input
            value={imdbId}
            onChange={(e) => setImdbId(e.target.value)}
            placeholder="e.g. tt1234567"
          />
        </div>
      </ModalContent>
      <ModalFooter>
        <Button variant="secondary" onClick={handleClose}>Cancel</Button>
        <Button onClick={handleSubmit} disabled={isPending || !imdbId}>
          {isPending ? 'Saving...' : 'Save'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
