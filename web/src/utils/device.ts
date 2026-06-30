/**
 * Device Detection Utilities
 * Provides functions to detect specific device types and browser capabilities
 */

/**
 * Checks if the current browser is an LG WebOS TV environment.
 * Evaluates core platform strings safely across modern, legacy,
 * and specific version-locked webOS Chromium engines.
 */
export const isWebOSTV = (): boolean => {
  if (typeof navigator === 'undefined') {
    return false
  }

  const ua = navigator.userAgent
  return /WebOS|webOS/i.test(ua) || /LG Browser/i.test(ua)
}

/**
 * Helper to identify legacy/resource-constrained WebOS environments
 * (like Chromium 79 or below) requiring heavy optimization features.
 */
export const isLegacyWebOS = (): boolean => {
  if (typeof navigator === 'undefined') {
    return false
  }

  const ua = navigator.userAgent
  const match = ua.match(/Chrome\/(\d+)/)

  if (match && match[1]) {
    const chromeVersion = parseInt(match[1], 10)
    return isWebOSTV() && chromeVersion <= 79
  }

  return false
}

/**
 * Checks if the current browser is Chromium 79 (or compatible)
 * Used for feature detection of legacy browser support
 */
export const isChromium79 = (): boolean => {
  if (typeof navigator === 'undefined') {
    return false
  }

  const ua = navigator.userAgent
  return /Chrome\/79/.test(ua) || /Chromium\/79/.test(ua)
}

/**
 * Checks if Fullscreen API is available
 * Chrome 79 requires user gesture for requestFullscreen()
 */
interface FullscreenDocumentElement extends HTMLElement {
  webkitRequestFullscreen?: () => Promise<void>
  mozRequestFullScreen?: () => Promise<void>
  msRequestFullscreen?: () => Promise<void>
}

interface FullscreenDocument extends Document {
  webkitExitFullscreen?: () => Promise<void>
  mozCancelFullScreen?: () => Promise<void>
  msExitFullscreen?: () => Promise<void>
}

export const hasFullscreenAPI = (): boolean => {
  if (typeof document === 'undefined') {
    return false
  }

  const el = document.documentElement as FullscreenDocumentElement
  return !!(
    el.requestFullscreen ||
    el.webkitRequestFullscreen ||
    el.mozRequestFullScreen ||
    el.msRequestFullscreen
  )
}

/**
 * Checks if the browser supports the fullscreen API without user gesture
 * This is typically false for Chrome 79 on WebOS TV
 */
export const supportsAutomaticFullscreen = (): boolean => {
  if (isWebOSTV()) {
    return false
  }
  return true
}

/**
 * Gets the user preference for auto-fullscreen
 * Returns true by default for WebOS TV, false otherwise
 */
export const getAutoFullscreenPreference = (): boolean => {
  try {
    const stored = localStorage.getItem('auto-fullscreen')
    if (stored !== null) {
      return stored === 'true'
    }
  } catch {
    // localStorage not available
  }

  return true
}

/**
 * Sets the user preference for auto-fullscreen
 */
export const setAutoFullscreenPreference = (enabled: boolean): void => {
  try {
    localStorage.setItem('auto-fullscreen', String(enabled))
  } catch {
    // Ignore errors (private browsing, etc.)
  }
}

// Track active escape listeners to avoid structural memory leaks on CSS fullscreen toggles
const activeCssListeners = new Map<HTMLElement, (e: KeyboardEvent) => void>()

/**
 * Attempts to enter fullscreen mode for the video container
 * Falls back to CSS-based fullscreen if API fails
 *
 * On WebOS TV, skips the native Fullscreen API entirely to avoid the
 * browser's "Press ESC to exit full screen" overlay, which cannot be
 * suppressed. Uses CSS-based fullscreen instead.
 */
export const enterFullscreen = (element: HTMLElement): Promise<void> => {
  if (isWebOSTV()) {
    return enterCSSFullscreen(element)
  }

  const el = element as FullscreenDocumentElement

  if (el.requestFullscreen) {
    return el.requestFullscreen().catch(() => enterCSSFullscreen(element))
  }

  if (el.webkitRequestFullscreen) {
    return el.webkitRequestFullscreen().catch(() => enterCSSFullscreen(element))
  }

  if (el.mozRequestFullScreen) {
    return el.mozRequestFullScreen().catch(() => enterCSSFullscreen(element))
  }

  if (el.msRequestFullscreen) {
    return el.msRequestFullscreen().catch(() => enterCSSFullscreen(element))
  }

  return enterCSSFullscreen(element)
}

/**
 * Cleanly exits fullscreen mode handling both Native and CSS variants
 */
export const exitFullscreen = (element: HTMLElement): Promise<void> => {
  if (isInCSSFullscreen(element)) {
    exitCSSFullscreen(element)
    return Promise.resolve()
  }

  const doc = document as FullscreenDocument

  if (doc.exitFullscreen) {
    return doc.exitFullscreen()
  } else if (doc.webkitExitFullscreen) {
    return doc.webkitExitFullscreen()
  } else if (doc.mozCancelFullScreen) {
    return doc.mozCancelFullScreen()
  } else if (doc.msExitFullscreen) {
    return doc.msExitFullscreen()
  }

  return Promise.resolve()
}

/**
 * CSS-based fullscreen fallback for browsers where Fullscreen API
 * requires user gesture or is unavailable
 */
const enterCSSFullscreen = (element: HTMLElement): Promise<void> => {
  return new Promise((resolve) => {
    element.classList.add('fullscreen-fallback')
    document.body.style.overflow = 'hidden'

    // Cleanup any existing listeners attached to this element before assigning a new one
    if (activeCssListeners.has(element)) {
      const oldListener = activeCssListeners.get(element)
      if (oldListener) document.removeEventListener('keydown', oldListener)
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        exitCSSFullscreen(element)
        resolve()
      }
    }

    activeCssListeners.set(element, handleKeyDown)
    document.addEventListener('keydown', handleKeyDown)
  })
}

/**
 * Exits CSS-based fullscreen mode and detaches related listeners cleanly
 */
export const exitCSSFullscreen = (element: HTMLElement): void => {
  element.classList.remove('fullscreen-fallback')
  document.body.style.overflow = ''

  const handler = activeCssListeners.get(element)
  if (handler) {
    document.removeEventListener('keydown', handler)
    activeCssListeners.delete(element)
  }
}

/**
 * Checks if currently in CSS-based fullscreen mode
 */
export const isInCSSFullscreen = (element: HTMLElement): boolean => {
  return element.classList.contains('fullscreen-fallback')
}

/**
 * Checks if the device is a TV (WebOS, Tizen, etc.)
 * Used to apply TV-specific styles like hiding the cursor
 */
export const isTVDevice = (): boolean => {
  return isWebOSTV()
}

/**
 * Apply TV-specific styles when on a TV device
 * Call this on app initialization to hide cursor on TV
 */
export const applyTVStyles = (): void => {
  if (typeof document === 'undefined') {
    return
  }

  if (isTVDevice()) {
    document.body.classList.add('tv-device')
  }
}