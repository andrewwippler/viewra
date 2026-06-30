import { createFileRoute } from '@tanstack/react-router'
import { AdminRoute } from '@/components/common'
import { MissingFilesManager } from '@/features/nitpicky-edits'

export const Route = createFileRoute('/_layout/settings/nitpicky')({
  component: () => (
    <AdminRoute>
      <MissingFilesManager />
    </AdminRoute>
  ),
})
