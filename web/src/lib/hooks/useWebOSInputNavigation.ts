import { useEffect, useRef, useCallback } from 'react'
import { isWebOSTV } from '@/utils/device'

export interface UseWebOSInputNavigationOptions {
  enabled?: boolean
  onFocusChange?: (element: HTMLElement | null) => void
  onBackPress?: () => void
  /** CSS selector for row containers. When set, Up/Down navigates between rows and Left/Right within a row. */
  rowSelector?: string
}

const FOCUSABLE_SELECTOR = 'input:not([type="hidden"]):not([disabled]), button:not([disabled]), a[href], area[href], object, select:not([disabled]), textarea:not([disabled]), [contenteditable]'

const isVisible = (element: Element): boolean => {
  const el = element as HTMLElement
  return el.offsetWidth > 0 && el.offsetHeight > 0
}

const navigationInit = (): void => {
  const connectEl = document.querySelector('#connect')
  if (connectEl instanceof HTMLElement && isVisible(connectEl)) {
    connectEl.focus()
    return
  }

  const abortEl = document.querySelector('#abort')
  if (abortEl instanceof HTMLElement && isVisible(abortEl)) {
    abortEl.focus()
    return
  }

  const firstFocusable = document.querySelector(FOCUSABLE_SELECTOR) as HTMLElement | null
  if (firstFocusable && isVisible(firstFocusable)) {
    firstFocusable.focus()
  }
}

const isNativeInteractive = (el: Element): el is HTMLElement => {
  if (!(el instanceof HTMLElement)) {
    return false
  }

  const tag = el.tagName
  return tag === 'BUTTON' || tag === 'A' || tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA'
}

const getFocusableInRow = (row: HTMLElement): HTMLElement[] => {
  const elements = row.querySelectorAll(FOCUSABLE_SELECTOR)
  return Array.from(elements).filter((el): el is HTMLElement => isNativeInteractive(el) || isVisible(el))
}

export const useWebOSInputNavigation = (options: UseWebOSInputNavigationOptions = {}) => {
  const { enabled = true, onFocusChange, onBackPress, rowSelector } = options
  const isActiveRef = useRef(false)

  const navigate = useCallback((direction: 'up' | 'down' | 'left' | 'right') => {
    const element = document.activeElement

    if (!element || element === document.body || element.tagName === 'BODY') {
      navigationInit()
      return
    }

    if (!isVisible(element)) {
      navigationInit()
      return
    }

    // Row-based navigation
    if (rowSelector) {
      const currentRow = element.closest(rowSelector) as HTMLElement | null

      if (direction === 'up' || direction === 'down') {
        if (!currentRow) {
          navigationInit()
          return
        }

        const allRows = Array.from(document.querySelectorAll(rowSelector)) as HTMLElement[]
        const rowIndex = allRows.indexOf(currentRow)

        if (rowIndex === -1) {
          navigationInit()
          return
        }

        const step = direction === 'up' ? -1 : 1
        let targetRowIndex = rowIndex + step
        while (targetRowIndex >= 0 && targetRowIndex < allRows.length) {
          const targetRow = allRows[targetRowIndex]
          const focusable = getFocusableInRow(targetRow)
          if (focusable.length > 0) {
            focusable[0].focus()
            focusable[0].scrollIntoView({ behavior: 'smooth', block: 'center' })
            if (onFocusChange) {
              onFocusChange(focusable[0])
            }
            break
          }
          targetRowIndex += step
        }
      } else if (direction === 'left' || direction === 'right') {
        if (!currentRow) {
          return
        }

        const focusable = getFocusableInRow(currentRow)
        if (focusable.length <= 1) {
          return
        }

        const currentIndex = focusable.indexOf(element as HTMLElement)
        if (currentIndex === -1) {
          return
        }

        let targetIndex: number
        if (direction === 'left') {
          targetIndex = currentIndex === 0 ? focusable.length - 1 : currentIndex - 1
        } else {
          targetIndex = currentIndex === focusable.length - 1 ? 0 : currentIndex + 1
        }

        if (focusable[targetIndex]) {
          focusable[targetIndex].focus()
          focusable[targetIndex].scrollIntoView({ behavior: 'smooth', block: 'nearest' })
          if (onFocusChange) {
            onFocusChange(focusable[targetIndex])
          }
        }
      }
      return
    }

    // Legacy linear navigation (no rowSelector)
    if (direction === 'left' || direction === 'right') {
      return
    }

    const allElements = document.querySelectorAll(FOCUSABLE_SELECTOR)
    const visibleElements = Array.from(allElements).filter((el): el is HTMLElement => isVisible(el))
    const currentIndex = visibleElements.indexOf(element as HTMLElement)

    if (currentIndex === -1) {
      navigationInit()
      return
    }

    const amount = direction === 'up' ? -1 : 1
    const targetIndex = currentIndex + amount

    if (visibleElements[targetIndex]) {
      visibleElements[targetIndex].focus()
      visibleElements[targetIndex].scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      })
      if (onFocusChange) {
        onFocusChange(visibleElements[targetIndex])
      }
    }
  }, [onFocusChange, rowSelector])

  const handleBack = useCallback(() => {
    if (onBackPress) {
      onBackPress()
    } else if (typeof window !== 'undefined' && 'webOS' in window && typeof (window as unknown as { webOS: { platformBack: () => void } }).webOS?.platformBack === 'function') {
      ;(window as unknown as { webOS: { platformBack: () => void } }).webOS.platformBack()
    } else {
      window.history.back()
    }
  }, [onBackPress])

  useEffect(() => {
    const shouldActivate = enabled

    if (!shouldActivate) {
      return
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.keyCode) {
        case 38: // Up arrow
          e.preventDefault()
          e.stopPropagation()
          navigate('up')
          break
        case 40: // Down arrow
          e.preventDefault()
          e.stopPropagation()
          navigate('down')
          break
        case 37: // Left arrow
          e.preventDefault()
          e.stopPropagation()
          navigate('left')
          break
        case 39: // Right arrow
          e.preventDefault()
          e.stopPropagation()
          navigate('right')
          break
        case 461: // Back button on WebOS remote
          e.preventDefault()
          e.stopPropagation()
          handleBack()
          break
      }
    }

    document.addEventListener('keydown', handleKeyDown, true)
    isActiveRef.current = true

    if (document.activeElement === document.body || !document.activeElement) {
      setTimeout(() => {
        navigationInit()
      }, 100)
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown, true)
      isActiveRef.current = false
    }
  }, [enabled, onFocusChange, onBackPress, navigate, handleBack])

  return {
    isActive: isActiveRef.current,
    navigate,
    navigationInit,
  }
}

export const useWebOSKeyboardScroll = () => {
  useEffect(() => {
    if (!isWebOSTV()) {
      return
    }

    const handleFocus = (e: FocusEvent) => {
      const target = e.target as HTMLElement

      if (
        !(
          target instanceof HTMLInputElement ||
          target instanceof HTMLTextAreaElement ||
          target.getAttribute('contenteditable') === 'true'
        )
      ) {
        return
      }

      setTimeout(() => {
        const inputRect = target.getBoundingClientRect()
        const viewportHeight = window.innerHeight
        const keyboardHeight = viewportHeight * 0.4

        const inputBottom = inputRect.bottom
        const visibleArea = viewportHeight - keyboardHeight

        if (inputBottom > visibleArea) {
          const scrollOffset = inputBottom - visibleArea + 20

          window.scrollTo({
            top: window.scrollY + scrollOffset,
            behavior: 'smooth',
          })
        }
      }, 300)
    }

    document.addEventListener('focusin', handleFocus, true)

    return () => {
      document.removeEventListener('focusin', handleFocus, true)
    }
  }, [])
}

export const TVFocusStyles = {
  focused: 'tv-focused',
  inputFocused: 'tv-input-focused',
  formGroup: 'tv-form-group',
} as const

export const TVFocusCSS = `
.tv-focused {
  outline: 3px solid var(--color-primary-500, #3b82f6);
  outline-offset: 2px;
}

.tv-input-focused {
  outline: 3px solid var(--color-primary-500, #3b82f6);
  outline-offset: 2px;
  box-shadow: 0 0 0 6px var(--color-primary-500/20, rgba(59, 130, 246, 0.2));
}

.tv-form-group:focus-within {
  outline: 2px solid var(--color-primary-400, #60a5fa);
  outline-offset: 1px;
  border-radius: 8px;
}

.tv-focused,
.tv-input-focused,
.tv-form-group:focus-within {
  transition: outline 150ms ease-out, box-shadow 150ms ease-out;
}
`
