import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { App } from './App'

// Set TV mode flag on <html> for CSS scoping.
// In TV build (__TV_MODE__ = true), [data-tv] enables TV focus outlines.
// In web build, [data-tv] is absent so TV styles are suppressed.
if (__TV_MODE__) { document.documentElement.setAttribute('data-tv', '') }

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>
)
