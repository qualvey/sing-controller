import type * as THREE from 'three/webgpu'
import type { EarthProjection, EarthView } from './projection'
import type { EarthEndpointInfo, EarthLocation, EarthRoute } from './types'

export type { EarthProjection, EarthView }
export type EarthColorScheme = 'dark' | 'light'
export type EarthVisualMode = 'flat' | 'space' | 'color'

export interface EarthViewState {
  quaternion: [number, number, number, number]
  distance: number
}

export interface EarthRendererOptions {
  reducedMotion: boolean
  projection: EarthProjection
  visualMode: EarthVisualMode
  colorScheme: EarthColorScheme
  initialViewState?: EarthViewState
  onViewStateChange?: (state: EarthViewState) => void
  onEndpointHover: (info: EarthEndpointInfo | null, x?: number, y?: number) => void
}

export interface EarthRenderer {
  setRoutes: (routes: EarthRoute[]) => void
  setInitialLocation: (location: EarthLocation) => void
  setReducedMotion: (reduced: boolean) => void
  setAutoRotation: (enabled: boolean) => void
  setCityLabelsVisible: (visible: boolean) => void
  setProjection: (projection: EarthProjection) => void
  setVisualMode: (mode: EarthVisualMode) => void
  setColorScheme: (scheme: EarthColorScheme) => void
  getViewState: () => EarthViewState
  setViewState: (state: EarthViewState) => void
  dispose: () => void
}

export interface EarthRenderEndpoint extends EarthEndpointInfo {
  key: string
  position: THREE.Vector3
  latitude: number
  longitude: number
  direct?: boolean
}

export interface EarthRenderSnapshot {
  signature: string
  routes: readonly EarthRoute[]
  endpoints: readonly EarthRenderEndpoint[]
}

