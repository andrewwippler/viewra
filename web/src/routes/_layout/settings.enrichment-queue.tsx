import { createFileRoute } from '@tanstack/react-router'
import { AdminRoute, SettingsPage } from '@/components/common'
import { EnrichmentQueueManager } from '@/features/enrichment-queue'

export const Route = createFileRoute('/_layout/settings/enrichment-queue')({
  component: () => (
    <AdminRoute>
      <SettingsPage>
        <SettingsPage.Header
          title="Enrichment Queue"
          description="View and manage metadata enrichment jobs across all libraries."
        />
        <EnrichmentQueueManager />
      </SettingsPage>
    </AdminRoute>
  ),
})
