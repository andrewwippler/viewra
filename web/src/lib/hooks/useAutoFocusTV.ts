import { useEffect } from 'react'

const isVisible = (el: Element) =>
  (el as HTMLElement).offsetWidth > 0 && (el as HTMLElement).offsetHeight > 0

const SELECTORS = [
  'button, a, input, [tabindex]:not([tabindex="-1"]), [role="button"]',
]

export const useAutoFocusTV = (deps: unknown[] = []) => {
  useEffect(() => {
    const id = requestAnimationFrame(() => {
      for (const sel of SELECTORS) {
        const candidates = document.querySelectorAll<HTMLElement>(sel)
        for (const el of candidates) {
          if (isVisible(el)) {
            el.focus()
            return
          }
        }
      }
    })
    return () => cancelAnimationFrame(id)
  }, deps) // eslint-disable-line react-hooks/exhaustive-deps
}
