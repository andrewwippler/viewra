/**
 * TV Input Module
 * 
 * Provides spatial navigation, focus management, and input handling
 * optimized for TV platforms (webOS, Tizen, etc.)
 */

export { SpatialNavigation, createSpatialNavigation } from './SpatialNavigation'
export type { SpatialNavigationOptions, Direction, FocusableElement } from './SpatialNavigation'

export { TvInputProvider, useTvInput, useSpatialNavigation } from './TvInputProvider'