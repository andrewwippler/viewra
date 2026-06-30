import { useNavigate } from '@tanstack/react-router'
import { Card, CardHeader, CardContent, Button } from '@/components/ui'
import { text } from '@/styles/semantic'
import { cn } from '@/lib/utils'
import { Trash2 } from 'lucide-react'

const StorageCleanupCard = () => {
  const navigate = useNavigate()

  return (
    <Card variant="glass">
      <CardHeader className="border-b border-neutral-100 dark:border-neutral-800">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={cn(
                'p-2 rounded-lg',
                'bg-neutral-100 dark:bg-neutral-800',
                'text-neutral-600 dark:text-neutral-400'
              )}
            >
              <Trash2 className="w-5 h-5" />
            </div>
            <div>
              <h2 className={cn('text-lg font-semibold', text.primary)}>Storage Cleanup</h2>
              <p className={cn('text-sm mt-0.5', text.secondary)}>
                Find and remove orphaned media files
              </p>
            </div>
          </div>
        </div>
      </CardHeader>

      <CardContent className="py-4">
        <div className="flex items-center justify-between">
          <div className="flex-1 mr-4">
            <p className={cn('text-sm', text.secondary)}>
              Scan your libraries for media files that no longer exist on disk and remove their
              database entries to keep your library clean.
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate({ to: '/settings/nitpicky' })}
            className="shrink-0"
          >
            <Trash2 className="w-4 h-4 mr-1.5" />
            Storage Cleanup
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export { StorageCleanupCard }
