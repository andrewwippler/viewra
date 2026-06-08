/**
 * Device Detection Utilities
 * Provides functions to detect specific device types and browser capabilities
 */

/**
 * Checks if the current browser is a WebOS TV (Chromium 79 based)
 * WebOS TV user agents typically contain "WebOS" and "Chrome/79"
 */
export function isWebOSTV(): boolean {
  if (typeof navigator === 'undefined') return false;
  
  const ua = navigator.userAgent;
  return /WebOS/.test(ua) && /Chrome\/79/.test(ua);
}

/**
 * Checks if the current browser is Chromium 79 (or compatible)
 * Used for feature detection of legacy browser support
 */
export function isChromium79(): boolean {
  if (typeof navigator === 'undefined') return false;
  
  const ua = navigator.userAgent;
  return /Chrome\/79/.test(ua) || /Chromium\/79/.test(ua);
}

/**
 * Checks if Fullscreen API is available
 * Chrome 79 requires user gesture for requestFullscreen()
 */
export function hasFullscreenAPI(): boolean {
  if (typeof document === 'undefined') return false;
  
  return !!(
    document.documentElement.requestFullscreen ||
    (document.documentElement as any).webkitRequestFullscreen ||
    (document.documentElement as any).mozRequestFullScreen ||
    (document.documentElement as any).msRequestFullscreen
  );
}

/**
 * Checks if the browser supports the fullscreen API without user gesture
 * This is typically false for Chrome 79 on WebOS TV
 */
export function supportsAutomaticFullscreen(): boolean {
  // WebOS TV Chromium 79 requires user gesture
  if (isWebOSTV()) return false;
  
  // Modern browsers generally support it
  return true;
}

/**
 * Gets the user preference for auto-fullscreen
 * Returns true by default for WebOS TV, false otherwise
 */
export function getAutoFullscreenPreference(): boolean {
  try {
    const stored = localStorage.getItem('auto-fullscreen');
    if (stored !== null) {
      return stored === 'true';
    }
  } catch {
    // localStorage not available
  }
  
  // Default: true for WebOS TV, false for desktop
  return isWebOSTV();
}

/**
 * Sets the user preference for auto-fullscreen
 */
export function setAutoFullscreenPreference(enabled: boolean): void {
  try {
    localStorage.setItem('auto-fullscreen', String(enabled));
  } catch {
    // Ignore errors (private browsing, etc.)
  }
}

/**
 * Attempts to enter fullscreen mode for the video container
 * Falls back to CSS-based fullscreen if API fails
 */
export function enterFullscreen(element: HTMLElement): Promise<void> {
  // Try native Fullscreen API first
  if (element.requestFullscreen) {
    return element.requestFullscreen().catch(() => {
      // Fallback to CSS-based fullscreen
      return enterCSSFullscreen(element);
    });
  }
  
  // WebKit prefix (older Safari/iOS)
  if ((element as any).webkitRequestFullscreen) {
    return (element as any).webkitRequestFullscreen().catch(() => {
      return enterCSSFullscreen(element);
    });
  }
  
  // Firefox
  if ((element as any).mozRequestFullScreen) {
    return (element as any).mozRequestFullScreen().catch(() => {
      return enterCSSFullscreen(element);
    });
  }
  
  // IE/Edge
  if ((element as any).msRequestFullscreen) {
    return (element as any).msRequestFullscreen().catch(() => {
      return enterCSSFullscreen(element);
    });
  }
  
  // Fallback to CSS-based fullscreen
  return enterCSSFullscreen(element);
}

/**
 * CSS-based fullscreen fallback for browsers where Fullscreen API
 * requires user gesture or is unavailable
 */
function enterCSSFullscreen(element: HTMLElement): Promise<void> {
  return new Promise((resolve) => {
    element.classList.add('fullscreen-fallback');
    document.body.style.overflow = 'hidden';
    
    // Listen for escape key to exit
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        exitCSSFullscreen(element);
        document.removeEventListener('keydown', handleKeyDown);
        resolve();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
  });
}

/**
 * Exits CSS-based fullscreen mode
 */
export function exitCSSFullscreen(element: HTMLElement): void {
  element.classList.remove('fullscreen-fallback');
  document.body.style.overflow = '';
}

/**
 * Checks if currently in CSS-based fullscreen mode
 */
export function isInCSSFullscreen(element: HTMLElement): boolean {
  return element.classList.contains('fullscreen-fallback');
}/**
 * Checks if the device is a TV (WebOS, Tizen, etc.)
 * Used to apply TV-specific styles like hiding the cursor
 */
export function isTVDevice(): boolean {
  return isWebOSTV()
}

/**
 * Apply TV-specific styles when on a TV device
 * Call this on app initialization to hide cursor on TV
 */
export function applyTVStyles(): void {
  if (typeof document === 'undefined') return
  
  if (isTVDevice()) {
    document.body.classList.add('tv-device')
  }
}
