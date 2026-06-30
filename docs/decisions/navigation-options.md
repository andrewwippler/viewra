# TV Navigation Options

Options considered for implementing D-pad/arrow-key spatial navigation on WebOS TV and other TV platforms.

## Problem

Users navigate our TV UI with a remote control D-pad. Arrow keys must move focus between cards, buttons, and other interactive elements. The navigation must work on WebOS (Chrome 79), which has specific rendering quirks.

---

## Option 1: `useEffect` + `useRef` + `SpatialNavigation` class (CURRENT/INITIAL)

A `useEffect` with `[]` deps in the layout calls `createSpatialNavigation(container, { selector: '.tv-focusable' })`. The class listens for `keydown` on `document`, filters by `root.contains(e.target)`, and moves focus using a distance/score algorithm.

**Problems:**
- The effect runs before auth resolves — `containerRef.current` is `null` on first render → early return → never re-runs → SpatialNavigation never created
- `root.contains(e.target)` rejects events when nothing has focus (target is `document`/`body`)
- No detection of dynamically added elements (cards load asynchronously)
- `.sn-focused` CSS only targets `.focusable`, not `.tv-focusable`

---

## Option 2: Callback ref pattern (CHOSEN)

Replace `useRef` with `useCallback` ref. The callback fires **synchronously** when the DOM node mounts/unmounts, regardless of auth timing. No effect dependency needed.

**Implementation:**
- Store SpatialNavigation instance + cleanup in a `useRef`
- Callback ref creates SN on mount, runs cleanup on unmount
- Also fix: relax event target filter, add MutationObserver for dynamic content, add CSS

**Pros:**
- No timing dependency — works on first render, after auth, after navigation
- Clean lifecycle — no stale effects
- Simple to implement (one callback function)
- Chosen approach

**Cons:**
- Slightly less idiomatic React (callback refs are less common than useEffect)

---

## Option 3: Add `isAuthenticated` to `useEffect` deps

Keep the existing `useEffect` pattern but add `isAuthenticated` to the dependency array so it re-runs when auth resolves.

**Implementation:**
```ts
useEffect(() => {
  const container = containerRef.current
  if (!container) return
  const sn = createSpatialNavigation(container, { ... })
  return () => { sn.destroy() }
}, [isAuthenticated])
```

**Pros:**
- Minimal code change
- Familiar pattern

**Cons:**
- Effect still runs when auth is loading (first render) — early return is wasted work
- Race condition: effect might run before the div is in the DOM
- Doesn't fix the `root.contains()` filter or dynamic content issues (still need separate fixes)
- Need to handle effect cleanup + re-creation if the component doesn't unmount between auth states

---

## Option 4: Jellyfin-web command/event pattern

Adopt the jellyfin-web architecture: `keyboardNavigation.js` maps keycodes → commands, `inputManager.js` dispatches commands as custom DOM events, and components/focus managers listen for those events.

**Implementation sketch:**
```
keydown → keyboardNavigation maps to 'left'/'right'/'up'/'down'/'back'
       → dispatches CustomEvent('command', { detail: { command } }) on activeElement
       → bubbles up → focusManager or individual components handle it
```

**Pros:**
- Proven in production (jellyfin-web)
- Decoupled — any component can listen for commands
- No root container dependency — events bubble naturally
- Handles all key types (media keys, gamepad, back button)

**Cons:**
- Major refactor — would need to rewrite SpatialNavigation class and all consumers
- Over-engineered for current needs (only D-pad + back needed)
- Custom event dispatch adds indirection that makes debugging harder

---

## Option 5: Third-party library

Use an off-the-shelf spatial navigation library like:
- [`navigo`](https://github.com/jeffcmay/navigo) — lightweight spatial navigation
- [`spatial-navigation-js`](https://github.com/luke-chang/js-spatial-navigation) — full-featured, used by Firefox TV
- [`@noriginmedia/react-spatial-navigation`](https://github.com/NoriginMedia/react-spatial-navigation) — React-specific

**Pros:**
- Battle-tested
- Less code to maintain

**Cons:**
- Dependency overhead
- May not handle Chrome 79 quirks (logical properties, `:focus-visible`, etc.)
- Hard to customize for our specific UI patterns
- Bundle size increase

---

## Option 6: Native WebOS navigation

WebOS provides `webOS.service.request()` and the Luna Surface Manager for navigation. The `ares-webos-sdk` includes directional navigation support via `enyo`/`mojoloader`.

**Pros:**
- Native performance
- True platform integration

**Cons:**
- Only works on WebOS — Tizen, Fire TV, etc. need separate implementations
- Requires WebOS SDK to develop/test
- LG's Enyo/Mojo frameworks are deprecated
- Wouldn't work in a standard web browser for testing

---

## Decision

**Option 2** was chosen. It directly solves the initialization timing bug without adding unnecessary complexity or third-party dependencies. Combined with fixes to the event filter, MutationObserver, and CSS, it provides reliable spatial navigation across all TV platforms.
