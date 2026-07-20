import { createFileRoute } from '@tanstack/react-router'
import { AdminRoute, SettingsPage } from '@/components/common'
import { MissingFilesManager } from '@/features/nitpicky-edits'

export const Route = createFileRoute('/_layout/settings/nitpicky')({
  component: () => (
    <AdminRoute>
      <SettingsPage>
        <SettingsPage.Header
          title="Storage Cleanup"
          description="Scan your media directories to detect files that have been moved or deleted.
          Missing items can be removed from the database."
        />
        <MissingFilesManager />
      </SettingsPage>
    </AdminRoute>
  ),
})
