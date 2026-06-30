import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router'
import { useLayoutEffect } from 'react'
import { useAuth } from '@/contexts'
import { useWebOSKeyboardScroll } from '@/lib/hooks'

const WebosLayout = () => {
  const navigate = useNavigate()
  const { isAuthenticated, isLoading, needsSetup } = useAuth()

  useWebOSKeyboardScroll()

  useLayoutEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate({ to: needsSetup ? '/setup' : '/login' })
    }
  }, [isAuthenticated, isLoading, needsSetup, navigate])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-neutral-950">
        <div className="animate-pulse text-neutral-500">Loading...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return null
  }

  return (
    <div className="h-full bg-neutral-950">
      <Outlet />
    </div>
  )
}

export const Route = createFileRoute('/webos')({
  component: WebosLayout,
})
