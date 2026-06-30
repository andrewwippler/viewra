import { useState, useRef, useEffect, useCallback } from 'react'
import { cn } from '@/lib/utils'
import { videoOverlay } from '@/styles/semantic'
import type { DropdownBaseProps } from './DropdownBase.types'

export const DropdownBase = ({
  buttonContent,
  icon,
  minButtonWidth = '80px',
  ariaLabel,
  children,
  panelWidth = 'w-56',
}: DropdownBaseProps) => {
  const [showPanel, setShowPanel] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  const close = useCallback(() => setShowPanel(false), [])

  // Close panel when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        panelRef.current &&
        buttonRef.current &&
        !panelRef.current.contains(e.target as Node) &&
        !buttonRef.current.contains(e.target as Node)
      ) {
        setShowPanel(false)
      }
    }

    if (showPanel) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showPanel])

  // TV mode: keyboard navigation within dropdown panel
  useEffect(() => {
    if (!__TV_MODE__) {return}
    if (!showPanel) {return}

    const panel = panelRef.current
    if (!panel) {return}

    const getOptions = (): HTMLElement[] => {
      const options = panel.querySelectorAll<HTMLElement>('[role="option"]')
      return Array.from(options).filter((el) => {
        const rect = el.getBoundingClientRect()
        return rect.width > 0 && rect.height > 0
      })
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      const options = getOptions()
      if (options.length === 0) {return}

      const currentIndex = options.indexOf(document.activeElement as HTMLElement)

      switch (e.keyCode) {
        case 40: // ArrowDown
        case 39: // ArrowRight
          e.preventDefault()
          e.stopPropagation()
          {
            const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % options.length
            options[nextIndex].focus()
          }
          break
        case 38: // ArrowUp
        case 37: // ArrowLeft
          e.preventDefault()
          e.stopPropagation()
          {
            const nextIndex = currentIndex === -1 ? options.length - 1 : (currentIndex - 1 + options.length) % options.length
            options[nextIndex].focus()
          }
          break
        case 13: // Enter
          if (currentIndex >= 0) {
            e.preventDefault()
            e.stopPropagation()
            options[currentIndex].click()
          }
          break
        case 27: // Escape
          e.preventDefault()
          e.stopPropagation()
          close()
          if (buttonRef.current) {
            buttonRef.current.focus()
          }
          break
      }
    }

    document.addEventListener('keydown', handleKeyDown, { capture: true })
    // Focus first option when panel opens
    const firstOption = getOptions()[0]
    if (firstOption) {
      firstOption.focus()
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown, { capture: true })
    }
  }, [showPanel, close])

  return (
    <div className="relative">
      {/* Dropdown button */}
      <button
        ref={buttonRef}
        onClick={() => setShowPanel(!showPanel)}
        className={cn(
          'text-xs sm:text-sm rounded-md px-2 sm:px-3 py-1.5 cursor-pointer flex items-center gap-1',
          videoOverlay.patterns.button,
          videoOverlay.ring.focus
        )}
        style={{ minWidth: minButtonWidth }}
        aria-label={ariaLabel}
        aria-expanded={showPanel}
        aria-haspopup="listbox"
      >
        {icon}
        <span>{buttonContent}</span>
        {/* Chevron */}
        <svg
          className={cn('w-3 h-3 transition-transform', showPanel && 'rotate-180')}
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {/* Dropdown panel */}
      {showPanel && (
        <div
          ref={panelRef}
          className={cn(
            'absolute bottom-full right-0 mb-2 rounded-lg overflow-hidden z-50',
            panelWidth,
            videoOverlay.patterns.panel
          )}
          role="listbox"
        >
          {children({ close })}
        </div>
      )}
    </div>
  )
}
