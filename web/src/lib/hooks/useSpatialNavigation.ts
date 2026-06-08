/**
 * useSpatialNavigation Hook
 * 
 * React hook for integrating spatial navigation with TV remote controls.
 * Provides automatic focus management and navigation for interactive elements.
 */

import { useEffect, useRef, useState, useCallback } from 'react'
import { createSpatialNavigation, type SpatialNavigation, type SpatialNavigationOptions } from '@/lib/tv'

interface UseSpatialNavigationOptions extends SpatialNavigationOptions {
  /** Whether to auto-activate on mount (default: true on TV devices) */
  autoActivate?: boolean
  /** Callback when focus changes */
  onFocusChange?: (element: HTMLElement | null) => void
}

interface UseSpatialNavigationReturn {
  /** Ref to attach to the container element */
  containerRef: React.RefObject<HTMLElement | null>
  /** Whether spatial navigation is currently active */
  isActive: boolean
  /** The currently focused element */
  focusedElement: HTMLElement | null
  /** Manually move focus in a direction */
  moveFocus: (direction: 'up' | 'down' | 'left' | 'right') => void
  /** Focus a specific element */
  focusElement: (element: HTMLElement) => void
  /** Focus an element by selector */
  focusBySelector: (selector: string) => boolean
  /** Activate the currently focused element */
  activateCurrent: () => void
  /** Clean up navigation instance */
  destroy: () => void
}

/**
 * Hook for using spatial navigation in React components
 */
export function useSpatialNavigation(
  options: UseSpatialNavigationOptions = {}
): UseSpatialNavigationReturn {
  const {
    selector = '.focusable',
    autoFocus = true,
    debug = false,
    autoActivate = true,
    onFocusChange,
    ...snOptions
  } = options

  const containerRef = useRef<HTMLElement>(null)
  const navigationRef = useRef<SpatialNavigation | null>(null)
  const [isActive, setIsActive] = useState(false)
  const [focusedElement, setFocusedElement] = useState<HTMLElement | null>(null)

  // Initialize spatial navigation
  useEffect(() => {
    if (!autoActivate || !containerRef.current) return

    const container = containerRef.current
    
    navigationRef.current = createSpatialNavigation(container, {
      selector,
      autoFocus,
      debug,
      ...snOptions,
    })

    setIsActive(true)

    // Track focus changes
    const handleFocus = () => {
      const current = navigationRef.current?.getCurrentFocus() ?? null
      setFocusedElement(current)
      onFocusChange?.(current)
    }

    container.addEventListener('focusin', handleFocus)

    return () => {
      container.removeEventListener('focusin', handleFocus)
      navigationRef.current?.destroy()
      navigationRef.current = null
      setIsActive(false)
      setFocusedElement(null)
    }
  }, [autoActivate, selector, autoFocus, debug, onFocusChange, snOptions])

  // Move focus in a specific direction
  const moveFocus = useCallback((direction: 'up' | 'down' | 'left' | 'right') => {
    navigationRef.current?.moveFocus(direction)
  }, [])

  // Focus a specific element
  const focusElement = useCallback((element: HTMLElement) => {
    navigationRef.current?.focus(element as Parameters<typeof navigationRef.current.focus>[0])
  }, [])

  // Focus by selector
  const focusBySelector = useCallback((selector: string): boolean => {
    return navigationRef.current?.focusBySelector(selector) ?? false
  }, [])

  // Activate the currently focused element
  const activateCurrent = useCallback(() => {
    navigationRef.current?.activateCurrent()
  }, [])

  // Clean up
  const destroy = useCallback(() => {
    navigationRef.current?.destroy()
    navigationRef.current = null
    setIsActive(false)
    setFocusedElement(null)
  }, [])

  return {
    containerRef,
    isActive,
    focusedElement,
    moveFocus,
    focusElement,
    focusBySelector,
    activateCurrent,
    destroy,
  }
}

/**
 * useGridNavigation - Specialized hook for grid layouts
 */
export function useGridNavigation(
  options: {
    /** Number of columns in the grid */
    columns?: number
    /** Selector for grid items */
    selector?: string
    /** Callback when focus changes */
    onFocusChange?: (index: number) => void
  } = {}
) {
  const { columns = 4, selector = '.grid-item', onFocusChange } = options
  
  const { containerRef, isActive, focusedElement, moveFocus, destroy } = useSpatialNavigation({
    selector,
    autoFocus: true,
  })

  const currentIndexRef = useRef(-1)

  // Calculate grid navigation
  const handleMoveFocus = useCallback((direction: 'up' | 'down' | 'left' | 'right') => {
    if (!isActive || !containerRef.current) return

    const items = containerRef.current.querySelectorAll(selector)
    if (items.length === 0) return

    // Get current focused item index
    let currentIndex = currentIndexRef.current
    if (currentIndex < 0 && focusedElement) {
      currentIndex = Array.from(items).indexOf(focusedElement as HTMLElement)
    }

    let nextIndex = currentIndex

    switch (direction) {
      case 'up':
        nextIndex = currentIndex - columns
        break
      case 'down':
        nextIndex = currentIndex + columns
        break
      case 'left':
        nextIndex = Math.max(0, currentIndex - 1)
        break
      case 'right':
        nextIndex = Math.min(items.length - 1, currentIndex + 1)
        break
    }

    // Wrap around at edges
    if (nextIndex < 0) nextIndex = currentIndex
    if (nextIndex >= items.length) nextIndex = currentIndex

    const itemsArray: HTMLElement[] = Array.from(items) as HTMLElement[]
    if (nextIndex !== currentIndex && itemsArray[nextIndex]) {
      const itemToFocus = itemsArray[nextIndex]
      currentIndexRef.current = nextIndex
      itemToFocus.focus()
      onFocusChange?.(nextIndex)
    }
  }, [isActive, columns, selector, focusedElement, containerRef, onFocusChange])

  return {
    containerRef,
    isActive,
    focusedElement,
    moveFocus: handleMoveFocus,
    destroy,
  }
}