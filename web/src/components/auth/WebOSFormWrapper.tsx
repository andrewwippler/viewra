import { useWebOSInputNavigation } from '@/lib/hooks'

export const WebOSFormWrapper = ({ children, className }: { children: React.ReactNode; className?: string }) => {
  useWebOSInputNavigation()

  return <div className={className}>{children}</div>
}
