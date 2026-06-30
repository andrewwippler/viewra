import { createFileRoute } from '@tanstack/react-router'
import { TvHome } from '@/views/home/TvHome'

export const Route = createFileRoute('/webos/')({
  component: () => <TvHome />,
})
