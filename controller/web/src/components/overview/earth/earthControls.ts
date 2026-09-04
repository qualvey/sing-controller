import * as THREE from 'three/webgpu'
import type { EarthLocation } from './types'
import { toEarthVector } from './earthMath'
import type { EarthViewState } from './rendererTypes'

export interface EarthControlsOptions {
  camera: THREE.PerspectiveCamera
  domElement: HTMLElement
  earthGroup: THREE.Group
  initialViewState?: EarthViewState
  minDistance?: number
  maxDistance?: number
  rotateSpeed?: number
  zoomSpeed?: number
  inertiaDamping?: number
  autoRotate?: boolean
  autoRotateSpeed?: number
}

export class EarthControls {
  public camera: THREE.PerspectiveCamera
  public domElement: HTMLElement
  public earthGroup: THREE.Group

  public minDistance: number
  public maxDistance: number
  public rotateSpeed: number
  public zoomSpeed: number
  public inertiaDamping: number
  public autoRotate: boolean
  public autoRotateSpeed: number
  public target = new THREE.Vector3(0, 0, 0)

  private isDragging = false
  private activePointerId: number | null = null
  private lastTrackballVec = new THREE.Vector3()
  private angularVelocityAxis = new THREE.Vector3(0, 1, 0)
  private angularVelocity = 0

  private currentDistance: number
  private targetDistance: number

  // Touch pinch support
  private touches = new Map<number, { x: number; y: number }>()
  private prevPinchDistance = 0

  private listeners = new Map<string, Set<() => void>>()
  private cleanups: (() => void)[] = []

  constructor(options: EarthControlsOptions) {
    this.camera = options.camera
    this.domElement = options.domElement
    this.earthGroup = options.earthGroup

    this.minDistance = options.minDistance ?? 2.4
    this.maxDistance = options.maxDistance ?? 8.0
    this.rotateSpeed = options.rotateSpeed ?? 1.25
    this.zoomSpeed = options.zoomSpeed ?? 1.0
    this.inertiaDamping = options.inertiaDamping ?? 0.94
    this.autoRotate = options.autoRotate ?? true
    this.autoRotateSpeed = options.autoRotateSpeed ?? 0.5

    this.currentDistance = this.camera.position.distanceTo(this.target)
    this.targetDistance = this.currentDistance

    this.initEventListeners()

    if (options.initialViewState) {
      this.setViewState(options.initialViewState)
    }
  }

  public getViewState(): EarthViewState {
    return {
      quaternion: [
        this.earthGroup.quaternion.x,
        this.earthGroup.quaternion.y,
        this.earthGroup.quaternion.z,
        this.earthGroup.quaternion.w,
      ],
      distance: this.targetDistance,
    }
  }

  public setViewState(state: EarthViewState) {
    if (Array.isArray(state.quaternion) && state.quaternion.length === 4) {
      this.earthGroup.quaternion.fromArray(state.quaternion)
    }
    if (
      typeof state.distance === 'number' &&
      state.distance >= this.minDistance &&
      state.distance <= this.maxDistance
    ) {
      this.targetDistance = state.distance
      this.currentDistance = state.distance
      const dir = this.camera.position.clone().sub(this.target).normalize()
      this.camera.position.copy(this.target).addScaledVector(dir, this.currentDistance)
      this.camera.lookAt(this.target)
    }
    this.emitChange()
  }

  private getTrackballVector(clientX: number, clientY: number): THREE.Vector3 {
    const bounds = this.domElement.getBoundingClientRect()
    const width = Math.max(1, bounds.width)
    const height = Math.max(1, bounds.height)
    const minDim = Math.min(width, height)

    // Compute normalized coordinates centered at (0, 0)
    const u = (2 * (clientX - bounds.left) - width) / minDim
    const v = (height - 2 * (clientY - bounds.top)) / minDim

    const r2 = u * u + v * v
    let w = 0
    if (r2 <= 0.5) {
      w = Math.sqrt(Math.max(0, 1 - r2))
    } else {
      w = 0.5 / Math.sqrt(r2)
    }

    return new THREE.Vector3(u, v, w).normalize()
  }

  private initEventListeners() {
    const el = this.domElement

    const onPointerDown = (e: PointerEvent) => {
      if (e.button !== 0 && e.pointerType === 'mouse') return

      this.touches.set(e.pointerId, { x: e.clientX, y: e.clientY })

      if (this.touches.size === 1) {
        this.isDragging = true
        this.activePointerId = e.pointerId
        this.angularVelocity = 0
        this.lastTrackballVec = this.getTrackballVector(e.clientX, e.clientY)
        el.setPointerCapture?.(e.pointerId)
      } else if (this.touches.size === 2) {
        this.isDragging = false
        const [t1, t2] = Array.from(this.touches.values())
        this.prevPinchDistance = Math.hypot(t1.x - t2.x, t1.y - t2.y)
      }
    }

    const onPointerMove = (e: PointerEvent) => {
      if (!this.touches.has(e.pointerId)) return
      this.touches.set(e.pointerId, { x: e.clientX, y: e.clientY })

      if (this.touches.size === 2) {
        const [t1, t2] = Array.from(this.touches.values())
        const currentPinch = Math.hypot(t1.x - t2.x, t1.y - t2.y)
        if (this.prevPinchDistance > 0 && currentPinch > 0) {
          const ratio = currentPinch / this.prevPinchDistance
          const factor = 1 - (ratio - 1) * 1.5
          this.targetDistance = THREE.MathUtils.clamp(
            this.targetDistance * factor,
            this.minDistance,
            this.maxDistance,
          )
          this.emitChange()
        }
        this.prevPinchDistance = currentPinch
        return
      }

      if (!this.isDragging || e.pointerId !== this.activePointerId) return

      const currentVec = this.getTrackballVector(e.clientX, e.clientY)
      const dot = THREE.MathUtils.clamp(this.lastTrackballVec.dot(currentVec), -1, 1)
      const axisCam = new THREE.Vector3().crossVectors(this.lastTrackballVec, currentVec)
      const angle = Math.acos(dot) * this.rotateSpeed

      if (axisCam.lengthSq() > 1e-8 && angle > 1e-6) {
        axisCam.normalize()
        const axisWorld = axisCam.clone().transformDirection(this.camera.matrixWorld)
        const deltaQuat = new THREE.Quaternion().setFromAxisAngle(axisWorld, angle)

        this.earthGroup.quaternion.premultiply(deltaQuat)
        this.angularVelocityAxis.copy(axisWorld)
        this.angularVelocity = angle * 25 // scaled momentum estimate

        this.lastTrackballVec.copy(currentVec)
        this.emitChange()
      }
    }

    const onPointerUp = (e: PointerEvent) => {
      this.touches.delete(e.pointerId)
      if (e.pointerId === this.activePointerId) {
        this.isDragging = false
        this.activePointerId = null
        try {
          el.releasePointerCapture?.(e.pointerId)
        } catch {
          // ignore
        }
      }
      if (this.touches.size === 1) {
        const remaining = Array.from(this.touches.entries())[0]
        this.isDragging = true
        this.activePointerId = remaining[0]
        this.lastTrackballVec = this.getTrackballVector(remaining[1].x, remaining[1].y)
      }
    }

    const onWheel = (e: WheelEvent) => {
      e.preventDefault()
      const deltaFactor = e.deltaY > 0 ? 1.12 : 0.89
      this.targetDistance = THREE.MathUtils.clamp(
        this.targetDistance * deltaFactor,
        this.minDistance,
        this.maxDistance,
      )
      this.emitChange()
    }

    el.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerUp)
    el.addEventListener('wheel', onWheel, { passive: false })

    this.cleanups.push(() => {
      el.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('pointermove', onPointerMove)
      window.removeEventListener('pointerup', onPointerUp)
      window.removeEventListener('pointercancel', onPointerUp)
      el.removeEventListener('wheel', onWheel)
    })
  }

  public update(delta = 0.016) {
    // 1. Smooth zoom damping
    if (Math.abs(this.currentDistance - this.targetDistance) > 1e-4) {
      this.currentDistance = THREE.MathUtils.damp(
        this.currentDistance,
        this.targetDistance,
        14,
        delta,
      )
      const dir = this.camera.position.clone().sub(this.target).normalize()
      this.camera.position.copy(this.target).addScaledVector(dir, this.currentDistance)
      this.camera.lookAt(this.target)
      this.emitChange()
    }

    // 2. Inertia velocity damping on release
    if (!this.isDragging && Math.abs(this.angularVelocity) > 1e-4) {
      const stepAngle = this.angularVelocity * delta
      const deltaQuat = new THREE.Quaternion().setFromAxisAngle(
        this.angularVelocityAxis,
        stepAngle,
      )
      this.earthGroup.quaternion.premultiply(deltaQuat)
      this.angularVelocity *= Math.pow(this.inertiaDamping, delta * 60)
      this.emitChange()
    }

    // 3. Smooth auto rotation when idle
    if (!this.isDragging && Math.abs(this.angularVelocity) <= 1e-4 && this.autoRotate) {
      const autoAngle = this.autoRotateSpeed * 0.35 * delta
      const upVector = new THREE.Vector3(0, 1, 0)
      const autoQuat = new THREE.Quaternion().setFromAxisAngle(upVector, autoAngle)
      this.earthGroup.quaternion.premultiply(autoQuat)
      this.emitChange()
    }
  }

  public focusLocation(location: Pick<EarthLocation, 'latitude' | 'longitude'>) {
    const locVector = toEarthVector(location).normalize()
    // Align with camera view direction (looking from camera towards target)
    const viewDir = this.camera.position.clone().sub(this.target).normalize()
    const targetQuat = new THREE.Quaternion().setFromUnitVectors(locVector, viewDir)
    this.earthGroup.quaternion.copy(targetQuat)
    this.angularVelocity = 0
    this.emitChange()
  }

  public addEventListener(type: string, listener: () => void) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set())
    }
    this.listeners.get(type)!.add(listener)
  }

  public removeEventListener(type: string, listener: () => void) {
    this.listeners.get(type)?.delete(listener)
  }

  private emitChange() {
    this.listeners.get('change')?.forEach((fn) => fn())
  }

  public dispose() {
    this.cleanups.forEach((fn) => fn())
    this.cleanups = []
    this.listeners.clear()
    this.touches.clear()
  }
}
