import * as THREE from 'three/webgpu'
import type { EarthLocation } from './types'
import { toEarthVector } from './earthMath'
import type { EarthProjection } from './projection'
import type { EarthViewState } from './rendererTypes'

export interface EarthControlsOptions {
  camera: THREE.PerspectiveCamera
  domElement: HTMLElement
  earthGroup: THREE.Group
  projection?: EarthProjection
  initialViewState?: EarthViewState
  minDistance?: number
  maxDistance?: number
  rotateSpeed?: number
  zoomSpeed?: number
  inertiaDamping?: number
  autoRotate?: boolean
  autoRotateSpeed?: number
  realignDelay?: number
  realignDuration?: number
  maxPitch?: number
}

export class EarthControls {
  public camera: THREE.PerspectiveCamera
  public domElement: HTMLElement
  public earthGroup: THREE.Group

  public projection: EarthProjection = '3d'
  public enabled = true
  public minDistance: number
  public maxDistance: number
  public rotateSpeed: number
  public zoomSpeed: number
  public inertiaDamping: number
  public autoRotate: boolean
  public autoRotateSpeed: number
  public realignDelay: number
  public realignDuration: number
  public maxPitch: number
  public target = new THREE.Vector3(0, 0, 0)

  private isDragging = false
  private activePointerId: number | null = null
  private activeDragMode: 'constrained' | 'free' | null = null
  private lastPointerPos = new THREE.Vector2()
  private currentYaw = 0
  private currentPitch = 0
  private constrainedYawVelocity = 0
  private constrainedPitchVelocity = 0

  private lastTrackballVec = new THREE.Vector3()
  private angularVelocityAxis = new THREE.Vector3(0, 1, 0)
  private angularVelocity = 0

  private realignTimer = 0
  private isRealigning = false
  private realignProgress = 0
  private realignStartQuat = new THREE.Quaternion()
  private realignTargetQuat = new THREE.Quaternion()

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
    this.projection = options.projection ?? '3d'

    this.minDistance = options.minDistance ?? 2.4
    this.maxDistance = options.maxDistance ?? 8.0
    this.rotateSpeed = options.rotateSpeed ?? 1.25
    this.zoomSpeed = options.zoomSpeed ?? 1.0
    this.inertiaDamping = options.inertiaDamping ?? 0.94
    this.autoRotate = options.autoRotate ?? true
    this.autoRotateSpeed = options.autoRotateSpeed ?? 0.5
    this.realignDelay = options.realignDelay ?? 0.5
    this.realignDuration = options.realignDuration ?? 2.2
    this.maxPitch = options.maxPitch ?? Math.PI / 3

    this.currentDistance = this.camera.position.distanceTo(this.target)
    this.targetDistance = this.currentDistance

    this.initEventListeners()

    if (options.initialViewState) {
      this.setViewState(options.initialViewState)
    }
  }

  public setProjection(projection: EarthProjection) {
    this.projection = projection
    this.isDragging = false
    this.activePointerId = null
    this.activeDragMode = null
    this.angularVelocity = 0
    this.constrainedYawVelocity = 0
    this.constrainedPitchVelocity = 0
    this.realignTimer = 0
    this.isRealigning = false
    if (projection === '2d') {
      this.targetDistance = this.camera.position.z
      this.currentDistance = this.camera.position.z
    } else {
      this.targetDistance = this.camera.position.distanceTo(this.target)
      this.currentDistance = this.targetDistance
    }
  }

  public setDistanceBounds(min: number, max: number) {
    this.minDistance = min
    this.maxDistance = max
    this.targetDistance = THREE.MathUtils.clamp(this.targetDistance, min, max)
    this.currentDistance = THREE.MathUtils.clamp(this.currentDistance, min, max)
  }

  public setTargetDistance(dist: number) {
    this.targetDistance = dist
    this.currentDistance = dist
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
    this.angularVelocity = 0
    this.constrainedYawVelocity = 0
    this.constrainedPitchVelocity = 0
    this.realignTimer = 0
    this.isRealigning = false
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
      if (!this.enabled) return
      this.touches.set(e.pointerId, { x: e.clientX, y: e.clientY })
      this.realignTimer = 0
      this.isRealigning = false

      if (this.projection === '2d') {
        if (e.pointerType === 'mouse' && e.button !== 0) return
        if (this.touches.size === 1) {
          this.isDragging = true
          this.activePointerId = e.pointerId
          this.lastPointerPos.set(e.clientX, e.clientY)
          el.setPointerCapture?.(e.pointerId)
        } else if (this.touches.size === 2) {
          this.isDragging = false
          const [t1, t2] = Array.from(this.touches.values())
          this.prevPinchDistance = Math.hypot(t1.x - t2.x, t1.y - t2.y)
        }
        return
      }

      if (e.pointerType === 'mouse') {
        if (e.button === 0) {
          // Desktop Left-click: constrained Yaw & Pitch (North pole locked upright in view, zero Roll)
          this.isDragging = true
          this.activePointerId = e.pointerId
          this.activeDragMode = 'constrained'
          this.angularVelocity = 0
          this.constrainedYawVelocity = 0
          this.constrainedPitchVelocity = 0
          this.lastPointerPos.set(e.clientX, e.clientY)

          // Decompose current quaternion into pitch and yaw
          const worldNorth = new THREE.Vector3(0, 1, 0).applyQuaternion(this.earthGroup.quaternion)
          this.currentPitch = THREE.MathUtils.clamp(
            Math.atan2(worldNorth.z, worldNorth.y),
            -this.maxPitch,
            this.maxPitch,
          )
          const qPitchInv = new THREE.Quaternion()
            .setFromAxisAngle(new THREE.Vector3(1, 0, 0), this.currentPitch)
            .invert()
          const qYaw = qPitchInv.multiply(this.earthGroup.quaternion.clone())
          this.currentYaw = 2 * Math.atan2(qYaw.y, qYaw.w)

          el.setPointerCapture?.(e.pointerId)
        } else if (e.button === 2) {
          // Desktop Right-click: full omnidirectional trackball
          this.isDragging = true
          this.activePointerId = e.pointerId
          this.activeDragMode = 'free'
          this.angularVelocity = 0
          this.constrainedYawVelocity = 0
          this.constrainedPitchVelocity = 0
          this.lastTrackballVec = this.getTrackballVector(e.clientX, e.clientY)
          el.setPointerCapture?.(e.pointerId)
        }
        return
      }

      // Mobile / Touch interactions in 3D
      if (this.touches.size === 1) {
        this.isDragging = true
        this.activePointerId = e.pointerId
        this.activeDragMode = 'free'
        this.angularVelocity = 0
        this.constrainedYawVelocity = 0
        this.constrainedPitchVelocity = 0
        this.lastTrackballVec = this.getTrackballVector(e.clientX, e.clientY)
        el.setPointerCapture?.(e.pointerId)
      } else if (this.touches.size === 2) {
        this.isDragging = false
        this.activeDragMode = null
        const [t1, t2] = Array.from(this.touches.values())
        this.prevPinchDistance = Math.hypot(t1.x - t2.x, t1.y - t2.y)
      }
    }

    const onPointerMove = (e: PointerEvent) => {
      if (!this.enabled) return
      if (!this.touches.has(e.pointerId)) return
      this.touches.set(e.pointerId, { x: e.clientX, y: e.clientY })

      if (this.projection === '2d') {
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

        const dx = e.clientX - this.lastPointerPos.x
        const dy = e.clientY - this.lastPointerPos.y
        this.lastPointerPos.set(e.clientX, e.clientY)

        const bounds = this.domElement.getBoundingClientRect()
        const height = Math.max(1, bounds.height)
        const halfFov = THREE.MathUtils.degToRad(this.camera.fov) / 2
        const factor = (2 * Math.tan(halfFov) * this.camera.position.z) / height

        const panX = -dx * factor
        const panY = dy * factor

        this.camera.position.x += panX
        this.camera.position.y += panY
        this.target.x += panX
        this.target.y += panY
        this.emitChange()
        return
      }

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

      if (this.activeDragMode === 'constrained') {
        const dx = e.clientX - this.lastPointerPos.x
        const dy = e.clientY - this.lastPointerPos.y
        this.lastPointerPos.set(e.clientX, e.clientY)

        const bounds = this.domElement.getBoundingClientRect()
        const minDim = Math.min(bounds.width, bounds.height) || 400
        const radPerPx = ((Math.PI * 2) / minDim) * this.rotateSpeed * 0.75

        const deltaYaw = dx * radPerPx
        const deltaPitch = dy * radPerPx

        this.currentYaw += deltaYaw
        this.currentPitch = THREE.MathUtils.clamp(
          this.currentPitch + deltaPitch,
          -this.maxPitch,
          this.maxPitch,
        )

        const qPitch = new THREE.Quaternion().setFromAxisAngle(
          new THREE.Vector3(1, 0, 0),
          this.currentPitch,
        )
        const qYaw = new THREE.Quaternion().setFromAxisAngle(
          new THREE.Vector3(0, 1, 0),
          this.currentYaw,
        )
        this.earthGroup.quaternion.copy(qPitch).multiply(qYaw)

        // Momentum estimate for constrained inertia
        this.constrainedYawVelocity = deltaYaw * 30
        this.constrainedPitchVelocity = deltaPitch * 30

        this.emitChange()
        return
      }

      if (this.activeDragMode === 'free') {
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
    }

    const onPointerUp = (e: PointerEvent) => {
      this.touches.delete(e.pointerId)
      if (e.pointerId === this.activePointerId) {
        this.isDragging = false
        this.activePointerId = null
        this.activeDragMode = null
        try {
          el.releasePointerCapture?.(e.pointerId)
        } catch {
          // ignore
        }
      }
      if (this.projection === '2d') {
        if (this.touches.size === 1) {
          const remaining = Array.from(this.touches.entries())[0]
          this.isDragging = true
          this.activePointerId = remaining[0]
          this.lastPointerPos.set(remaining[1].x, remaining[1].y)
        }
        return
      }
      if (this.touches.size === 1) {
        const remaining = Array.from(this.touches.entries())[0]
        this.isDragging = true
        this.activePointerId = remaining[0]
        this.activeDragMode = 'free'
        this.lastTrackballVec = this.getTrackballVector(remaining[1].x, remaining[1].y)
      }
    }

    const onWheel = (e: WheelEvent) => {
      if (!this.enabled) return
      e.preventDefault()
      this.realignTimer = 0
      this.isRealigning = false
      const deltaFactor = e.deltaY > 0 ? 1.12 : 0.89
      this.targetDistance = THREE.MathUtils.clamp(
        this.targetDistance * deltaFactor,
        this.minDistance,
        this.maxDistance,
      )
      this.emitChange()
    }

    const onContextMenu = (e: MouseEvent) => {
      e.preventDefault()
    }

    el.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerUp)
    el.addEventListener('wheel', onWheel, { passive: false })
    el.addEventListener('contextmenu', onContextMenu)

    this.cleanups.push(() => {
      el.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('pointermove', onPointerMove)
      window.removeEventListener('pointerup', onPointerUp)
      window.removeEventListener('pointercancel', onPointerUp)
      el.removeEventListener('wheel', onWheel)
      el.removeEventListener('contextmenu', onContextMenu)
    })
  }

  public update(delta = 0.016) {
    if (!this.enabled) return

    if (this.projection === '2d') {
      const isZooming = Math.abs(this.camera.position.z - this.targetDistance) > 1e-4
      if (isZooming) {
        this.camera.position.z = THREE.MathUtils.damp(
          this.camera.position.z,
          this.targetDistance,
          14,
          delta,
        )
        this.currentDistance = this.camera.position.z
        this.emitChange()
      }
      return
    }

    // 1. Smooth zoom damping in 3D
    const isZooming = Math.abs(this.currentDistance - this.targetDistance) > 1e-4
    if (isZooming) {
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
    const hasFreeInertia = !this.isDragging && Math.abs(this.angularVelocity) > 0.01
    if (hasFreeInertia) {
      const stepAngle = this.angularVelocity * delta
      const deltaQuat = new THREE.Quaternion().setFromAxisAngle(
        this.angularVelocityAxis,
        stepAngle,
      )
      this.earthGroup.quaternion.premultiply(deltaQuat)
      this.angularVelocity *= Math.pow(this.inertiaDamping, delta * 60)
      if (Math.abs(this.angularVelocity) <= 0.01) {
        this.angularVelocity = 0
      }
      this.emitChange()
    } else if (!this.isDragging && this.angularVelocity !== 0) {
      this.angularVelocity = 0
    }

    const hasConstrainedInertia =
      !this.isDragging &&
      (Math.abs(this.constrainedYawVelocity) > 0.01 ||
        Math.abs(this.constrainedPitchVelocity) > 0.01)

    if (hasConstrainedInertia) {
      this.currentYaw += this.constrainedYawVelocity * delta
      this.currentPitch = THREE.MathUtils.clamp(
        this.currentPitch + this.constrainedPitchVelocity * delta,
        -this.maxPitch,
        this.maxPitch,
      )
      const qPitch = new THREE.Quaternion().setFromAxisAngle(
        new THREE.Vector3(1, 0, 0),
        this.currentPitch,
      )
      const qYaw = new THREE.Quaternion().setFromAxisAngle(
        new THREE.Vector3(0, 1, 0),
        this.currentYaw,
      )
      this.earthGroup.quaternion.copy(qPitch).multiply(qYaw)

      this.constrainedYawVelocity *= Math.pow(this.inertiaDamping, delta * 60)
      this.constrainedPitchVelocity *= Math.pow(this.inertiaDamping, delta * 60)
      if (Math.abs(this.constrainedYawVelocity) <= 0.01) this.constrainedYawVelocity = 0
      if (Math.abs(this.constrainedPitchVelocity) <= 0.01) this.constrainedPitchVelocity = 0
      this.emitChange()
    } else if (!this.isDragging) {
      if (this.constrainedYawVelocity !== 0) this.constrainedYawVelocity = 0
      if (this.constrainedPitchVelocity !== 0) this.constrainedPitchVelocity = 0
    }

    const hasInertia = hasFreeInertia || hasConstrainedInertia

    // 3. User interaction or active motion resets idle timer & cancels realignment
    if (this.isDragging || hasInertia || isZooming) {
      this.realignTimer = 0
      this.isRealigning = false
      return
    }

    if (!this.autoRotate) {
      return
    }

    // 4. Idle countdown and smooth upright realignment before self-rotation
    if (!this.isRealigning) {
      // Check if Earth's North pole is already pointing straight up in world space
      const worldNorth = new THREE.Vector3(0, 1, 0).applyQuaternion(this.earthGroup.quaternion)
      const isUpright = worldNorth.y > 0.999

      if (isUpright) {
        this.realignTimer += delta
        if (this.realignTimer >= this.realignDelay) {
          const autoAngle = this.autoRotateSpeed * 0.35 * delta
          const upVector = new THREE.Vector3(0, 1, 0)
          const autoQuat = new THREE.Quaternion().setFromAxisAngle(upVector, autoAngle)
          this.earthGroup.quaternion.premultiply(autoQuat)
          this.emitChange()
        }
      } else {
        this.realignTimer += delta
        if (this.realignTimer >= this.realignDelay) {
          this.startRealignment()
        }
      }
    } else {
      // Execute upright realignment slerp
      this.realignProgress = Math.min(1, this.realignProgress + delta / this.realignDuration)
      // Cubic ease-in-out
      const t = this.realignProgress
      const easeT = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2

      this.earthGroup.quaternion.slerpQuaternions(
        this.realignStartQuat,
        this.realignTargetQuat,
        easeT,
      )
      this.emitChange()

      if (this.realignProgress >= 1) {
        this.isRealigning = false
        this.realignTimer = 0
      }
    }
  }

  private startRealignment() {
    this.isRealigning = true
    this.realignProgress = 0
    this.realignStartQuat.copy(this.earthGroup.quaternion)

    const camDir = this.camera.position.clone().sub(this.target).normalize()
    // Find what direction on the local Earth is currently facing the camera
    const invQuat = this.earthGroup.quaternion.clone().invert()
    const localCamDir = camDir.clone().applyQuaternion(invQuat)

    // Project localCamDir onto Earth's equatorial plane (XZ)
    const localH = new THREE.Vector3(localCamDir.x, 0, localCamDir.z)
    if (localH.lengthSq() < 1e-6) {
      localH.set(1, 0, 0)
    } else {
      localH.normalize()
    }

    // Camera view horizontal direction in world space
    const camH = new THREE.Vector3(camDir.x, 0, camDir.z)
    if (camH.lengthSq() < 1e-6) {
      camH.set(0, 0, 1)
    } else {
      camH.normalize()
    }

    // Compute rotation angle around world Y-axis (0, 1, 0) to align localH with camH
    const sinPhi = localH.z * camH.x - localH.x * camH.z
    const cosPhi = localH.x * camH.x + localH.z * camH.z
    const phi = Math.atan2(sinPhi, cosPhi)

    this.realignTargetQuat.setFromAxisAngle(new THREE.Vector3(0, 1, 0), phi)
  }

  public focusLocation(location: Pick<EarthLocation, 'latitude' | 'longitude'>) {
    if (this.projection === '2d') {
      this.target.set(0, 0, 0)
      this.camera.position.x = 0
      this.camera.position.y = 0
      this.emitChange()
      return
    }
    const locVector = toEarthVector(location).normalize()
    // Align with camera view direction (looking from camera towards target)
    const viewDir = this.camera.position.clone().sub(this.target).normalize()
    const targetQuat = new THREE.Quaternion().setFromUnitVectors(locVector, viewDir)
    this.earthGroup.quaternion.copy(targetQuat)
    this.angularVelocity = 0
    this.constrainedYawVelocity = 0
    this.constrainedPitchVelocity = 0
    this.realignTimer = 0
    this.isRealigning = false
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
    this.isDragging = false
    this.activePointerId = null
    this.activeDragMode = null
    this.constrainedYawVelocity = 0
    this.constrainedPitchVelocity = 0
    this.isRealigning = false
    this.realignTimer = 0
  }
}
