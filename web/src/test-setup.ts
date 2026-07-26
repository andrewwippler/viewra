import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Mock navigator.mediaSession
class MockMediaMetadata {
  title?: string
  artist?: string
  artwork?: unknown[]
  constructor(init?: MediaMetadataInit) {
    Object.assign(this, init)
  }
}
Object.defineProperty(window, 'MediaMetadata', {
  value: MockMediaMetadata,
  writable: true,
})
Object.defineProperty(navigator, 'mediaSession', {
  value: {
    metadata: null as MockMediaMetadata | null,
    setActionHandler: vi.fn(),
  },
  writable: true,
})

// Mock ResizeObserver
class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
Object.defineProperty(window, 'ResizeObserver', {
  value: ResizeObserver,
  writable: true,
})

// Mock IntersectionObserver
class IntersectionObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
}
Object.defineProperty(window, 'IntersectionObserver', {
  value: IntersectionObserver,
  writable: true,
})

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
  writable: true,
})

// Suppress console.error in tests (React act warnings, etc.)
const originalError = console.error
console.error = (...args: unknown[]) => {
  if (typeof args[0] === 'string' && args[0].includes('Warning:')) {
    return
  }
  originalError.call(console, ...args)
}
