/**
 * Spatial Navigation for TV Remote Controls
 * 
 * Provides D-pad navigation support for webOS and other TV platforms.
 * Maps directional remote keys to focus movement between interactive elements.
 */

export interface SpatialNavigationOptions {
  /** Selector for focusable elements (default: '.focusable') */
  selector?: string
  /** Whether to auto-focus the first focusable element on init */
  autoFocus?: boolean
  /** Focus trap within a container */
  trapFocus?: boolean
  /** Visual debug mode */
  debug?: boolean
}

export interface FocusableElement extends HTMLElement {
  dataset: HTMLElement['dataset'] & {
    snGroup?: string
    snDisable?: string
    snFocus?: string
  }
}

/**
 * Direction mapping for D-pad navigation
 */
export const DIRECTION_MAP = {
  ArrowUp: 'up',
  ArrowDown: 'down',
  ArrowLeft: 'left',
  ArrowRight: 'right',
} as const

export type Direction = (typeof DIRECTION_MAP)[keyof typeof DIRECTION_MAP]

/**
 * SpatialNavigation class for handling TV remote navigation
 */
export class SpatialNavigation {
  private root: HTMLElement
  private selector: string
  private currentFocus: FocusableElement | null = null
  private autoFocus: boolean
  private trapFocus: boolean
  private debug: boolean
  private boundKeyHandler: (e: KeyboardEvent) => void

  constructor(root: HTMLElement, options: SpatialNavigationOptions = {}) {
    this.root = root
    this.selector = options.selector || '.focusable'
    this.autoFocus = options.autoFocus ?? true
    this.trapFocus = options.trapFocus ?? false
    this.debug = options.debug ?? false
    this.boundKeyHandler = this.handleKeyDown.bind(this)
    
    this.init()
  }

  private init(): void {
    this.root.setAttribute('data-sn-enabled', 'true')
    
    // Add click handler to track focus manually
    this.root.addEventListener('click', this.handleClick.bind(this))
    
    // Attach keyboard handler
    document.addEventListener('keydown', this.boundKeyHandler)
    
    // Auto-focus first element if enabled
    if (this.autoFocus) {
      requestAnimationFrame(() => this.focusFirst())
    }
  }

  private handleClick(e: MouseEvent): void {
    const target = e.target as HTMLElement
    const focusable = target.closest(this.selector) as FocusableElement | null
    
    if (focusable && this.isFocusable(focusable)) {
      this.setFocus(focusable)
    }
  }

  private handleKeyDown(e: KeyboardEvent): void {
    // Only handle if event originates from within our root
    if (!this.root.contains(e.target as HTMLElement)) return
    
    const direction = DIRECTION_MAP[e.key as keyof typeof DIRECTION_MAP]
    
    if (direction) {
      e.preventDefault()
      e.stopPropagation()
      this.moveFocus(direction)
      return
    }
    
    // Enter key - trigger click on focused element
    if (e.key === 'Enter') {
      e.preventDefault()
      this.activateCurrent()
      return
    }
    
    // Escape key - handle back navigation
    if (e.key === 'Escape' || e.key === 'Backspace') {
      e.preventDefault()
      this.handleBack()
      return
    }
  }

  public moveFocus(direction: Direction): void {
    const candidates = this.getCandidates()
    if (candidates.length === 0) return

    const currentRect = this.currentFocus?.getBoundingClientRect()
    if (!currentRect) {
      this.focusFirst()
      return
    }

    const currentCenter = {
      x: currentRect.left + currentRect.width / 2,
      y: currentRect.top + currentRect.height / 2,
    }

    let bestCandidate: FocusableElement | null = null
    let bestScore = Infinity

    for (const candidate of candidates) {
      if (candidate === this.currentFocus) continue
      if (candidate.dataset.snDisable === 'true') continue

      const candidateRect = candidate.getBoundingClientRect()
      const candidateCenter = {
        x: candidateRect.left + candidateRect.width / 2,
        y: candidateRect.top + candidateRect.height / 2,
      }

      const dx = candidateCenter.x - currentCenter.x
      const dy = candidateCenter.y - currentCenter.y

      // Calculate direction score based on the desired direction
      let directionScore = 0
      switch (direction) {
        case 'up':
          if (dy >= 0) directionScore = 1000 // Penalize downward movement
          directionScore += dy
          break
        case 'down':
          if (dy <= 0) directionScore = 1000 // Penalize upward movement
          directionScore -= dy
          break
        case 'left':
          if (dx >= 0) directionScore = 1000 // Penalize rightward movement
          directionScore += dx
          break
        case 'right':
          if (dx <= 0) directionScore = 1000 // Penalize leftward movement
          directionScore -= dx
          break
      }

      // Euclidean distance for tie-breaking
      const distance = Math.sqrt(dx * dx + dy * dy)
      const totalScore = directionScore + distance * 0.1

      if (directionScore < 1000 && totalScore < bestScore) {
        bestScore = totalScore
        bestCandidate = candidate
      }
    }

    if (bestCandidate) {
      this.setFocus(bestCandidate)
      bestCandidate.scrollIntoView({ block: 'nearest', inline: 'nearest' })
    }
  }

  private getCandidates(): FocusableElement[] {
    return Array.from(this.root.querySelectorAll(this.selector)) as FocusableElement[]
  }

  private isFocusable(element: HTMLElement): boolean {
    if (element.dataset.snDisable === 'true') return false
    if (element.hasAttribute('disabled')) return false
    if (element.getAttribute('tabindex') === '-1') return false
    return true
  }

  private setFocus(element: FocusableElement): void {
    if (this.currentFocus) {
      this.currentFocus.classList.remove('sn-focused')
      this.currentFocus.removeAttribute('data-sn-active')
    }
    
    this.currentFocus = element
    element.classList.add('sn-focused')
    element.setAttribute('data-sn-active', 'true')
    
    // Focus the element for accessibility
    element.focus({ preventScroll: true })
    
    this.log('Focus moved to:', element)
  }

  private focusFirst(): void {
    const candidates = this.getCandidates()
    const firstFocusable = candidates.find(c => this.isFocusable(c))
    if (firstFocusable) {
      this.setFocus(firstFocusable)
    }
  }

  public activateCurrent(): void {
    if (this.currentFocus) {
      this.currentFocus.click()
      this.log('Activated:', this.currentFocus)
    }
  }

  private handleBack(): void {
    // Dispatch custom back event that can be handled by the app
    const backEvent = new CustomEvent('tvback', {
      bubbles: true,
      cancelable: true,
      detail: { source: 'SpatialNavigation' }
    })
    this.root.dispatchEvent(backEvent)
  }

  private log(...args: unknown[]): void {
    if (this.debug) {
      console.log('[SpatialNavigation]', ...args)
    }
  }

  /**
   * Get the currently focused element
   */
  public getCurrentFocus(): FocusableElement | null {
    return this.currentFocus
  }

  /**
   * Programmatically focus an element
   */
  public focus(element: FocusableElement): void {
    if (this.isFocusable(element)) {
      this.setFocus(element)
    }
  }

  /**
   * Focus by selector
   */
  public focusBySelector(selector: string): boolean {
    const element = this.root.querySelector(selector) as FocusableElement | null
    if (element && this.isFocusable(element)) {
      this.setFocus(element)
      return true
    }
    return false
  }

  /**
   * Clean up event listeners
   */
  public destroy(): void {
    document.removeEventListener('keydown', this.boundKeyHandler)
    this.root.removeAttribute('data-sn-enabled')
    this.currentFocus?.classList.remove('sn-focused')
    this.currentFocus = null
  }
}

/**
 * Create a spatial navigation instance for a container
 */
export function createSpatialNavigation(
  root: HTMLElement,
  options?: SpatialNavigationOptions
): SpatialNavigation {
  return new SpatialNavigation(root, options)
}
