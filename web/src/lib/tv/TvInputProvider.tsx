/**
 * TV Input Provider
 * 
 * Provides a context for TV-specific input handling including spatial navigation,
 * on-screen keyboard management, and focus state tracking.
 */

import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { createSpatialNavigation, type SpatialNavigation, type SpatialNavigationOptions } from './SpatialNavigation'
import { detectDeviceType } from '@/lib/capabilities'

interface TvInputContextValue {
  /** Whether the current device is a TV */
  isTv: boolean
  /** Whether spatial navigation is active */
  isNavigationActive: boolean
  /** The current focused element ID (if any) */
  focusedElementId: string | null
  /** Scroll the page up when keyboard appears (for webOS) */
  scrollForKeyboard: () => void
}

const TvInputContext = createContext<TvInputContextValue>({
  isTv: false,
  isNavigationActive: false,
  focusedElementId: null,
  scrollForKeyboard: () => {},
})

interface TvInputProviderProps {
  children: ReactNode
  options?: SpatialNavigationOptions
}

/**
 * Provider component for TV-specific input handling
 */
export function TvInputProvider({ children, options }: TvInputProviderProps) {
  const [isTv] = useState(() => detectDeviceType() === 'tv')
  const [isNavigationActive, setIsNavigationActive] = useState(false)
  const [focusedElementId, setFocusedElementId] = useState<string | null>(null)
  const navigationRef = useRef<SpatialNavigation | null>(null)
  const containerRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    // Only activate spatial navigation on TV devices
    if (!isTv) return

    const initNavigation = () => {
      // Find the main app container
      const appRoot = document.getElementById('root') || document.body
      
      if (appRoot && !navigationRef.current) {
        navigationRef.current = createSpatialNavigation(appRoot, {
          selector: options?.selector || '.focusable',
          autoFocus: options?.autoFocus ?? true,
          debug: options?.debug ?? false,
        })
        setIsNavigationActive(true)
      }
    }

    // Small delay to ensure DOM is ready
    const timeoutId = setTimeout(initNavigation, 100)

    return () => {
      clearTimeout(timeoutId)
      navigationRef.current?.destroy()
      navigationRef.current = null
      setIsNavigationActive(false)
    }
  }, [isTv, options])

  // Track focus changes
  useEffect(() => {
    if (!navigationRef.current) return

    const handleFocus = (e: FocusEvent) => {
      const target = e.target as HTMLElement
      setFocusedElementId(target.id || target.className.split(' ')[0] || null)
    }

    document.addEventListener('focusin', handleFocus)
    return () => document.removeEventListener('focusin', handleFocus)
  }, [])

  // Handle webOS keyboard - scroll view when input is focused
  const scrollForKeyboard = () => {
    // On webOS, the native keyboard slides up from the bottom
    // and consumes ~35-40% of the screen. We need to scroll
    // the active input into the upper portion of the screen.
    const activeElement = document.activeElement as HTMLElement
    if (activeElement && (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA')) {
      // Calculate offset to keep input in upper 40% of screen
      const scrollOffset = window.innerHeight * 0.4
      const elementTop = activeElement.getBoundingClientRect().top
      
      if (elementTop > scrollOffset) {
        window.scrollTo({
          top: window.scrollY + (elementTop - scrollOffset),
          behavior: 'smooth'
        })
      }
    }
  }

  // Handle input focus for keyboard scroll
  useEffect(() => {
    if (!isTv) return

    const handleInputFocus = () => {
      // Delay to allow keyboard animation to start
      setTimeout(scrollForKeyboard, 300)
    }

    const inputs = document.querySelectorAll('input, textarea')
    inputs.forEach(input => {
      input.addEventListener('focus', handleInputFocus)
    })

    return () => {
      inputs.forEach(input => {
        input.removeEventListener('focus', handleInputFocus)
      })
    }
  }, [isTv])

  return (
    <TvInputContext.Provider value={{
      isTv,
      isNavigationActive,
      focusedElementId,
      scrollForKeyboard,
    }}>
      {children}
    </TvInputContext.Provider>
  )
}

/**
 * Hook to access TV input context
 */
export function useTvInput(): TvInputContextValue {
  return useContext(TvInputContext)
}

/**
 * Hook to access spatial navigation instance
 */
export function useSpatialNavigation(): SpatialNavigation | null {
  // This would need to be provided via context if needed
  // For now, we just expose the isNavigationActive state
  const { isNavigationActive } = useContext(TvInputContext)
  return isNavigationActive ? {} as SpatialNavigation : null
}