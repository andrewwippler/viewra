import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import path from 'path'
import fs from 'fs'
import { execSync } from 'child_process'

const uiMode = process.env.VITE_UI_MODE || 'web'
const isTV = uiMode === 'tv'

const apiTarget = process.env.VITE_API_URL || 'http://localhost:8080'

// Get git short SHA for version display
let appVersion = 'dev'
try {
  appVersion = execSync('git rev-parse --short HEAD', { encoding: 'utf-8' }).trim()
} catch {
  // Fallback if not in a git repo
}

const fixNullishCoalescing = (code: string) => {
  let out = ''
  let lastEnd = 0
  let i = 0
  while (i < code.length) {
    if (code[i] === '?' && code[i + 1] === '?') {
      let j = i - 1
      while (j >= 0 && code[j] === ' ') {j--}
      let done = false
      while (!done && j >= 0) {
        if (code[j] === ']') {
          let depth = 1; j--
          while (j >= 0 && depth > 0) {
            if (code[j] === ']') {depth++}
            else if (code[j] === '[') {depth--}
            if (depth > 0) {j--}
          }
          j--
        } else if (code[j] === '.') {
          j--
        } else if (/[$\w]/.test(code[j])) {
          while (j >= 0 && /[$\w]/.test(code[j])) {j--}
        } else {
          done = true
        }
      }
      const lhsStart = j + 1
      const lhs = code.slice(lhsStart, i)

      out += code.slice(lastEnd, lhsStart)

      const isAssign = code[i + 2] === '='
      const opLen = isAssign ? 3 : 2

      let k = i + opLen
      let paren = 0, brack = 0, brace = 0
      let inStr = false, strCh: string|null = null, inTmpl = false
      while (k < code.length) {
        const ch = code[k]
        if (inStr) {
          if (ch === '\\') { k += 2; continue }
          if (ch === strCh) {inStr = false}
          k++; continue
        }
        if (inTmpl) {
          if (ch === '\\') { k += 2; continue }
          if (ch === '`') {inTmpl = false}
          k++; continue
        }
        if (ch === '"' || ch === "'") { inStr = true; strCh = ch; k++; continue }
        if (ch === '`') { inTmpl = true; k++; continue }
        if (ch === '(') { paren++; k++; continue }
        if (ch === ')') { if (!paren) {break;} paren--; k++; continue }
        if (ch === '[') { brack++; k++; continue }
        if (ch === ']') { if (!brack) {break;} brack--; k++; continue }
        if (ch === '{') { brace++; k++; continue }
        if (ch === '}') { if (!brace) {break;} brace--; k++; continue }
        if (!paren && !brack && !brace) {
          if (ch === ',' || ch === ';' || ch === ')') {break}
        }
        k++
      }
      const rhs = code.slice(i + opLen, k)

      if (isAssign) {
        out += `(${lhs}==null&&(${lhs}=${rhs}),${lhs})`
      } else {
        out += `${lhs}!=null?${lhs}:${rhs}`
      }

      lastEnd = k
      i = k
    } else {
      i++
    }
  }
  out += code.slice(lastEnd)
  return out
}

const plugins = [
  react(),
  TanStackRouterVite(),
  {
    name: 'remove-crossorigin',
    transformIndexHtml(html: string) {
      return html.replace(/\s*crossorigin\b/g, '')
    },
  },
]

const splitCSSValue = (value: string): string[] => {
  const parts: string[] = []
  let current = ''
  let paren = 0
  for (const ch of value) {
    if (ch === '(') { paren++ }
    else if (ch === ')') { paren-- }
    if (ch === ' ' && paren === 0) {
      if (current) { parts.push(current); current = '' }
    } else {
      current += ch
    }
  }
  if (current) { parts.push(current) }
  return parts
}

const stripWhere = (code: string): string => {
  let out = ''
  let i = 0
  while (i < code.length) {
    if (code.slice(i, i + 7) === ':where(') {
      let depth = 1
      let j = i + 7
      while (j < code.length && depth > 0) {
        if (code[j] === '(') { depth++ }
        else if (code[j] === ')') { depth-- }
        j++
      }
      out += code.slice(i + 7, j - 1)
      i = j
    } else {
      out += code[i]
      i++
    }
  }
  return out
}

if (isTV) {
  plugins.push(
    {
      name: 'strip-at-layer',
      transform(code: string, id: string) {
        if (!id.endsWith('.css')) {return}
        let result = ''
        let i = 0
        let depth = 0
        let current = ''
        let inLayer = false
        const LAYER_RE = /^@layer\s+\w+\s*\{/
        while (i < code.length) {
          if (!inLayer) {
            const match = code.slice(i).match(LAYER_RE)
            if (match) {
              result += current
              current = ''
              inLayer = true
              depth = 1
              i += match[0].length
              continue
            }
          }
          if (inLayer) {
            if (code[i] === '{') {depth++}
            else if (code[i] === '}') {
              depth--
              if (depth === 0) {
                inLayer = false
                i++
                continue
              }
            }
            current += code[i]
          } else {
            current += code[i]
          }
          i++
        }
        result += current
        return { code: result }
      },
    },
    {
      name: 'remove-optional-chaining',
      closeBundle() {
        const dir = path.resolve(__dirname, `dist/${uiMode}/assets`)
        if (!fs.existsSync(dir)) {return}
        for (const f of fs.readdirSync(dir)) {
          if (!f.endsWith('.js')) {continue}
          const fp = path.join(dir, f)
          let code = fs.readFileSync(fp, 'utf-8')
          const before = code

          code = code.replace(/(\w+)\?\.\(\)/g, (_, v) => `typeof ${v}=='function'&&${v}()`)
          code = code.replace(/\?\.(\d)/g, '? .$1')

          code = fixNullishCoalescing(code)

          if (code !== before) {
            fs.writeFileSync(fp, code)
            console.warn(`[remove-optional-chaining] patched ${f}`)
          }
        }
      },
    },
    {
      name: 'fix-chrome79-css',
      closeBundle() {
        const dir = path.resolve(__dirname, `dist/${uiMode}/assets`)
        if (!fs.existsSync(dir)) {return}
        for (const f of fs.readdirSync(dir)) {
          if (!f.endsWith('.css')) {continue}
          const fp = path.join(dir, f)
          let code = fs.readFileSync(fp, 'utf-8')
          const before = code

          code = code.replace(/inset\s*:\s*(.+?)(\s*!important)?\s*(;|})/g, (match, value, important, terminator) => {
            const imp = important || ''
            const parts = splitCSSValue(value.trim())
            switch (parts.length) {
              case 1: return `top:${parts[0]}${imp};right:${parts[0]}${imp};bottom:${parts[0]}${imp};left:${parts[0]}${imp}${terminator}`
              case 2: return `top:${parts[0]}${imp};right:${parts[1]}${imp};bottom:${parts[0]}${imp};left:${parts[1]}${imp}${terminator}`
              case 3: return `top:${parts[0]}${imp};right:${parts[1]}${imp};bottom:${parts[2]}${imp};left:${parts[1]}${imp}${terminator}`
              case 4: return `top:${parts[0]}${imp};right:${parts[1]}${imp};bottom:${parts[2]}${imp};left:${parts[3]}${imp}${terminator}`
              default: return match
            }
          })

          code = code.replace(/:focus-visible/g, ':focus')

          code = code.replace(/aspect-ratio:[^;}]+;?/g, '')

          code = stripWhere(code)
          code = code.replace(/:is\(/g, ':-webkit-any(')

          // Chrome 79 doesn't support logical properties — expand to physical
          code = code.replace(/(?:padding|margin)-(?:inline|block)\s*:\s*(.+?)(\s*!important)?\s*(;|})/g, (match, value, important, terminator) => {
            const imp = important || ''
            const parts = splitCSSValue(value.trim())
            const isInline = match.startsWith('padding-inline') || match.startsWith('margin-inline')
            const isPadding = match.startsWith('padding')
            const prop = isPadding ? 'padding' : 'margin'
            if (isInline) {
              if (parts.length === 1) { return `${prop}-left:${parts[0]}${imp};${prop}-right:${parts[0]}${imp}${terminator}` }
              return `${prop}-left:${parts[0]}${imp};${prop}-right:${parts[1]}${imp}${terminator}`
            }
            if (parts.length === 1) { return `${prop}-top:${parts[0]}${imp};${prop}-bottom:${parts[0]}${imp}${terminator}` }
            return `${prop}-top:${parts[0]}${imp};${prop}-bottom:${parts[1]}${imp}${terminator}`
          })

          // Chrome 79 doesn't support gap in flexbox — add margin fallbacks
          const gapFallbacks: string[] = []
          code.replace(/\.(gap-[^{]*)\{gap:([^;}]+)/g, (_m: string, cls: string, val: string) => {
            gapFallbacks.push(`.flex.${cls}>:not(:first-child){margin-left:${val}}`)
            gapFallbacks.push(`.flex.flex-col.${cls}>:not(:first-child){margin-top:${val}}`)
            return _m
          })
          code.replace(/\.(gap-x-[^{]*)\{column-gap:([^;}]+)/g, (_m: string, cls: string, val: string) => {
            gapFallbacks.push(`.flex.${cls}>:not(:first-child){margin-left:${val}}`)
            return _m
          })
          code.replace(/\.(gap-y-[^{]*)\{row-gap:([^;}]+)/g, (_m: string, cls: string, val: string) => {
            gapFallbacks.push(`.flex.flex-col.${cls}>:not(:first-child){margin-top:${val}}`)
            return _m
          })
          if (gapFallbacks.length) {
            code += gapFallbacks.join('')
            console.warn(`[fix-chrome79-css] gap fallback: ${gapFallbacks.length} rules generated`)
          }

          if (code !== before) {
            fs.writeFileSync(fp, code)
            console.warn(`[fix-chrome79-css] patched ${f}`)
          }
        }
      },
    },
  )
}

export default defineConfig({
  base: isTV ? '/webos/' : '/',
  define: {
    __TV_MODE__: isTV ? 'true' : 'false',
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  plugins,
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: `dist/${uiMode}`,
    target: isTV ? 'chrome79' : 'es2020',
  },
})
