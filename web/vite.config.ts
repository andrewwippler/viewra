import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import path from 'path'
import fs from 'fs'

// API target: use VITE_API_URL env var for Docker, default to localhost for local dev
const apiTarget = process.env.VITE_API_URL || 'http://localhost:8080'

// Strips ?? (nullish coalescing) and ??= (nullish assignment) from minified JS
// for Chromium 79 compatibility. Walks the code character-by-character to handle
// minified code where expressions have no whitespace.
function fixNullishCoalescing(code: string) {
  // Use source-range approach: scan for ??, then append segments from original code
  let out = ''
  let lastEnd = 0
  let i = 0
  while (i < code.length) {
    if (code[i] === '?' && code[i + 1] === '?') {
      // Walk backwards to find LHS start (handle .property and [property])
      let j = i - 1
      while (j >= 0 && code[j] === ' ') j--
      // Process LHS tokens right-to-left
      // Token types: identifier, .identifier, ]...[
      let done = false
      while (!done && j >= 0) {
        if (code[j] === ']') {
          // Bracket access [...]: find matching [
          let depth = 1; j--
          while (j >= 0 && depth > 0) {
            if (code[j] === ']') depth++
            else if (code[j] === '[') depth--
            if (depth > 0) j--
          }
          // Move past the [
          j--
        } else if (code[j] === '.') {
          // Property access
          j--
        } else if (/[$\w]/.test(code[j])) {
          // Identifier
          while (j >= 0 && /[$\w]/.test(code[j])) j--
        } else {
          done = true
        }
      }
      const lhsStart = j + 1
      const lhs = code.slice(lhsStart, i)

      // Append everything before the LHS (original code, unchanged)
      out += code.slice(lastEnd, lhsStart)

      const isAssign = code[i + 2] === '='
      const opLen = isAssign ? 3 : 2

      // Walk forward to find RHS end
      let k = i + opLen
      let paren = 0, brack = 0, brace = 0
      let inStr = false, strCh: string|null = null, inTmpl = false
      while (k < code.length) {
        const ch = code[k]
        if (inStr) {
          if (ch === '\\') { k += 2; continue }
          if (ch === strCh) inStr = false
          k++; continue
        }
        if (inTmpl) {
          if (ch === '\\') { k += 2; continue }
          if (ch === '`') inTmpl = false
          k++; continue
        }
        if (ch === '"' || ch === "'") { inStr = true; strCh = ch; k++; continue }
        if (ch === '`') { inTmpl = true; k++; continue }
        if (ch === '(') { paren++; k++; continue }
        if (ch === ')') { if (!paren) break; paren--; k++; continue }
        if (ch === '[') { brack++; k++; continue }
        if (ch === ']') { if (!brack) break; brack--; k++; continue }
        if (ch === '{') { brace++; k++; continue }
        if (ch === '}') { if (!brace) break; brace--; k++; continue }
        if (!paren && !brack && !brace) {
          if (ch === ',' || ch === ';' || ch === ')') break
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
  // Append any remaining code after last replacement
  out += code.slice(lastEnd)
  return out
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    TanStackRouterVite(),
    {
      name: 'remove-crossorigin',
      transformIndexHtml(html) {
        return html.replace(/\s*crossorigin\b/g, '')
      },
    },
    {
      name: 'strip-at-layer',
      transform(code, id) {
        if (!id.endsWith('.css')) return
        // Strip @layer name { ... } wrappers (unsupported in Chromium 79)
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
            if (code[i] === '{') depth++
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
        const dir = path.resolve(__dirname, 'dist/assets')
        if (!fs.existsSync(dir)) return
        for (const f of fs.readdirSync(dir)) {
          if (!f.endsWith('.js')) continue
          const fp = path.join(dir, f)
          let code = fs.readFileSync(fp, 'utf-8')
          const before = code

          // Strip optional chaining: func?.() → typeof func=='function'&&func()
          code = code.replace(/(\w+)\?\.\(\)/g, (_, v) => `typeof ${v}=='function'&&${v}()`)
          // Fix decimal after ?. (minifier produces "?.5" which must be "? .5")
          code = code.replace(/\?\.(\d)/g, '? .$1')

          // Strip nullish coalescing (??) and nullish assignment (??=)
          // Chromium 79 doesn't support either
          code = fixNullishCoalescing(code)

          if (code !== before) {
            fs.writeFileSync(fp, code)
            console.log(`[remove-optional-chaining] patched ${f}`)
          }
        }
      },
    },
  ],
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
    target: 'chrome79',
  },
})
