import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { CSS2DRenderer } from 'three/addons/renderers/CSS2DRenderer.js'
import * as THREE from 'three/webgpu'
import { createCityLabelLayer } from './cityLabelLayer'
import { getRealtimeSunDirection, toEarthVector } from './earthMath'
import { createEndpointLayer } from './endpointLayer'
import { createGlobeLayer } from './globeLayer'
import {
  easeMorph,
  PLANE_HALF_HEIGHT,
  PLANE_HALF_WIDTH,
  projectionMorph,
  toLocalSample,
  wrapLongitude,
  type EarthProjection,
  type EarthView,
} from './projection'
import { createEarthRenderSnapshot } from './renderSnapshot'
import type { EarthRenderer as EarthRendererContract, EarthRendererOptions } from './rendererTypes'
import { createRouteLayer } from './routeLayer'
import type { EarthLocation } from './types'

export type { EarthRenderer } from './rendererTypes'

const MAX_INITIAL_LATITUDE = 15
const ORBIT_MIN_DISTANCE = 2.65
const ORBIT_MAX_DISTANCE = 7.5
const MORPH_DURATION = 1.15
// A little breathing room so the map's edges are not flush with the viewport.
const MAP_FIT_MARGIN = 1.05
// The flat map starts zoomed in a touch past its fitted distance.
const MAP_DEFAULT_ZOOM = 1.2

const getTrackballVector = (clientX: number, clientY: number, domElement: HTMLElement): THREE.Vector3 => {
  const bounds = domElement.getBoundingClientRect()
  const width = Math.max(1, bounds.width)
  const height = Math.max(1, bounds.height)
  const minDim = Math.min(width, height)

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

type Cleanup = () => void

const runCleanups = (cleanups: Cleanup[]) => {
  let firstError: unknown

  while (cleanups.length > 0) {
    try {
      cleanups.pop()!()
    } catch (error) {
      firstError ??= error
    }
  }

  if (firstError) throw firstError
}

export const createEarthRenderer = async (
  container: HTMLElement,
  options: EarthRendererOptions,
): Promise<EarthRendererContract> => {
  const cleanups: Cleanup[] = []
  let disposed = false
  const registerCleanup = (cleanup: Cleanup) => cleanups.push(cleanup)
  const disposeResources = () => {
    if (disposed) return
    disposed = true
    runCleanups(cleanups)
  }

  const scene = new THREE.Scene()
  const camera = new THREE.PerspectiveCamera(28, 1, 0.1, 100)
  camera.position.set(3.7, 1.55, 3.2)

  const renderer = new THREE.WebGPURenderer({ alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.toneMapping = THREE.ACESFilmicToneMapping
  renderer.toneMappingExposure = 1.05
  renderer.domElement.className = 'h-full w-full cursor-grab active:cursor-grabbing'
  renderer.domElement.style.display = 'block'

  try {
    container.appendChild(renderer.domElement)
    registerCleanup(() => {
      renderer.setAnimationLoop(null)
      try {
        renderer.dispose()
      } finally {
        renderer.domElement.remove()
      }
    })
    await renderer.init()

    const labelRenderer = new CSS2DRenderer()
    labelRenderer.sortObjects = false
    labelRenderer.domElement.className = 'pointer-events-none absolute inset-0 overflow-hidden'
    labelRenderer.domElement.setAttribute('aria-hidden', 'true')
    container.appendChild(labelRenderer.domElement)
    registerCleanup(() => labelRenderer.domElement.remove())

    const controls = new OrbitControls(camera, renderer.domElement)
    controls.enableDamping = true
    controls.rotateSpeed = 0.55
    controls.zoomSpeed = 0.75
    // Keep browser scrolling/navigation gestures inside the canvas from competing
    // with OrbitControls on touch devices.
    renderer.domElement.style.touchAction = 'none'
    registerCleanup(() => controls.dispose())

    const earthGroup = new THREE.Group()
    scene.add(earthGroup)
    registerCleanup(() => scene.remove(earthGroup))

    // Everything that is not the globe itself lives here. The overlays do not
    // take part in the sphere/plane morph, so the transition simply hides this
    // group and rebuilds its contents in the target projection once it settles.
    const overlayGroup = new THREE.Group()
    earthGroup.add(overlayGroup)
    registerCleanup(() => earthGroup.remove(overlayGroup))

    let reducedMotion = options.reducedMotion
    let projection = options.projection
    let view: EarthView = { projection, centerLongitude: 0 }
    let visualMode = options.visualMode
    let colorScheme = options.colorScheme
    let autoRotation = true
    let initialLocationSet = false
    let pendingInitialLocation: EarthLocation | null = null
    let visible = !document.hidden
    let intersecting = true
    let pinnedEndpoint = false
    let currentSignature = ''
    let morph = projectionMorph(projection)
    let morphFrom = morph
    let morphTo = morph
    let morphElapsed = 0
    let morphing = false
    const quatFrom = new THREE.Quaternion()
    const quatTo = new THREE.Quaternion()
    const orbitQuat = new THREE.Quaternion()
    const orbitQOffset = new THREE.Quaternion()
    const cameraFrom = new THREE.Vector3()
    const cameraTo = new THREE.Vector3()
    // Panning the flat map moves the orbit target, which has to travel back to
    // the origin over the transition rather than snapping at the end of it.
    const targetFrom = new THREE.Vector3()
    const orbitCameraPosition = camera.position.clone()
    let orbitCenterLongitude = 0
    const clock = new THREE.Clock()
    const sunDirection = getRealtimeSunDirection()

    // Right-click omnidirectional trackball & realignment state (doc/earth.md)
    let isRightDragging = false
    let activeRightPointerId: number | null = null
    const lastTrackballVec = new THREE.Vector3()
    const angularVelocityAxis = new THREE.Vector3(0, 1, 0)
    let angularVelocity = 0
    const inertiaDamping = 0.94

    const realignDelay = 0.5
    const realignDuration = 2.2
    let realignTimer = 0
    let isRealigning = false
    let realignProgress = 0
    const realignStartQuat = new THREE.Quaternion()
    const realignTargetQuat = new THREE.Quaternion()
    const realignStartCamPos = new THREE.Vector3()
    const realignTargetCamPos = new THREE.Vector3()
    let isOrbitDragging = false

    if (options.initialViewState) {
      if (
        Array.isArray(options.initialViewState.quaternion) &&
        options.initialViewState.quaternion.length === 4
      ) {
        earthGroup.quaternion.fromArray(options.initialViewState.quaternion)
      }
      if (typeof options.initialViewState.distance === 'number') {
        const dir = camera.position.clone().sub(controls.target).normalize()
        camera.position.copy(controls.target).addScaledVector(dir, options.initialViewState.distance)
        controls.update()
      }
    }

    let viewStateTimer: number | null = null
    const emitViewStateChange = () => {
      if (options.onViewStateChange) {
        if (viewStateTimer !== null) window.clearTimeout(viewStateTimer)
        viewStateTimer = window.setTimeout(() => {
          if (!disposed) {
            options.onViewStateChange?.({
              quaternion: [
                earthGroup.quaternion.x,
                earthGroup.quaternion.y,
                earthGroup.quaternion.z,
                earthGroup.quaternion.w,
              ],
              distance: camera.position.distanceTo(controls.target),
            })
          }
        }, 300)
      }
    }
    controls.addEventListener('start', () => {
      isOrbitDragging = true
      realignTimer = 0
      isRealigning = false
    })
    controls.addEventListener('end', () => {
      isOrbitDragging = false
      realignTimer = 0
    })
    controls.addEventListener('change', () => {
      if (reducedMotion) render()
      emitViewStateChange()
    })
    registerCleanup(() => {
      if (viewStateTimer !== null) window.clearTimeout(viewStateTimer)
    })

    const globeLayer = await createGlobeLayer({
      scene,
      earthGroup,
      renderer,
      visualMode,
      colorScheme,
      sunDirection,
    })
    registerCleanup(() => globeLayer.dispose())

    const routeLayer = createRouteLayer({
      parent: overlayGroup,
      view,
      visualMode,
      colorScheme,
    })
    registerCleanup(() => routeLayer.dispose())

    const endpointLayer = createEndpointLayer({
      parent: overlayGroup,
      camera,
      view,
      visualMode,
      sunDirection,
    })
    registerCleanup(() => endpointLayer.dispose())

    const cityLabelLayer = createCityLabelLayer({
      parent: overlayGroup,
      camera,
      controls,
      labelRenderer,
      view,
    })
    registerCleanup(() => cityLabelLayer.dispose())

    // Distance at which the whole 2:1 map fits, whichever of the two axes is the
    // binding constraint for the current viewport.
    const mapDistance = () => {
      const halfFov = THREE.MathUtils.degToRad(camera.fov) / 2
      const fitHeight = PLANE_HALF_HEIGHT / Math.tan(halfFov)
      const fitWidth = PLANE_HALF_WIDTH / (Math.tan(halfFov) * camera.aspect)

      return Math.max(fitHeight, fitWidth) * MAP_FIT_MARGIN
    }

    // Where the camera rests when the flat map is entered or refitted.
    const mapViewDistance = () => mapDistance() / MAP_DEFAULT_ZOOM

    const applyControls = () => {
      if (projection === '2d') {
        const distance = mapDistance()

        controls.enableRotate = false
        controls.enablePan = true
        controls.screenSpacePanning = true
        controls.mouseButtons.LEFT = THREE.MOUSE.PAN
        controls.touches.ONE = THREE.TOUCH.PAN
        controls.touches.TWO = THREE.TOUCH.DOLLY_PAN
        controls.minDistance = distance * 0.35
        controls.maxDistance = distance * 1.15
      } else {
        controls.enableRotate = true
        controls.enablePan = false
        controls.screenSpacePanning = false
        controls.mouseButtons.LEFT = THREE.MOUSE.ROTATE
        controls.touches.ONE = THREE.TOUCH.ROTATE
        controls.touches.TWO = THREE.TOUCH.DOLLY_ROTATE
        controls.minDistance = ORBIT_MIN_DISTANCE
        controls.maxDistance = ORBIT_MAX_DISTANCE
      }
    }

    const applyView = () => {
      routeLayer.setView(view)
      endpointLayer.setView(view)
      cityLabelLayer.setView(view)
    }

    const applyInitialLocation = (location: EarthLocation) => {
      const distance = camera.position.distanceTo(controls.target)
      const local = toLocalSample(location, 1, view)
      const direction = toEarthVector({
        latitude: THREE.MathUtils.clamp(
          local.latitude,
          -MAX_INITIAL_LATITUDE,
          MAX_INITIAL_LATITUDE,
        ),
        longitude: local.longitude,
      })
        .applyQuaternion(earthGroup.quaternion)
        .normalize()

      camera.position.copy(controls.target).addScaledVector(direction, distance)
      controls.update()
    }

    const render = () => {
      if (!disposed && visible && intersecting) {
        cityLabelLayer.updateVisibility()
        renderer.render(scene, camera)
        labelRenderer.render(scene, camera)
      }
    }

    const finishMorph = () => {
      morphing = false
      morph = morphTo
      morphElapsed = 0
      globeLayer.setMorph(morph)

      if (projection === '3d') {
        view = { ...view, centerLongitude: orbitCenterLongitude }
        globeLayer.setCenterLongitude(view.centerLongitude)
        earthGroup.quaternion.copy(orbitQuat)
      } else {
        earthGroup.quaternion.copy(quatTo)
      }

      applyView()

      controls.target.set(0, 0, 0)
      applyControls()
      camera.position.copy(cameraTo)
      camera.lookAt(controls.target)
      controls.enabled = true
      controls.enableDamping = false
      controls.update()
      controls.enableDamping = true
      overlayGroup.visible = true

      if (projection === '3d' && pendingInitialLocation) {
        const location = pendingInitialLocation
        pendingInitialLocation = null
        applyInitialLocation(location)
      }
    }

    const advanceMorph = (delta: number) => {
      morphElapsed += delta

      const rawProgress = Math.min(1, morphElapsed / MORPH_DURATION)

      if (morphTo === 1) {
        // 3D -> 2D 展开管线（两阶段错峰动力学）：
        // 阶段 1（0 ~ 45% 时间）：球体姿态回正与相机环绕对齐。球体保持 100% 闭合（morph = 0），杜绝任何接缝外露！
        const splitPoint = 0.45
        if (rawProgress <= splitPoint) {
          const rotProgress = rawProgress / splitPoint
          const rotEased = easeMorph(rotProgress)
          earthGroup.quaternion.slerpQuaternions(quatFrom, quatTo, rotEased)

          const targetDist = THREE.MathUtils.lerp(cameraFrom.length(), cameraTo.length(), rotEased)
          camera.position.lerpVectors(cameraFrom, cameraTo, rotEased).setLength(targetDist)
          controls.target.copy(targetFrom).multiplyScalar(1 - rotEased)
          camera.lookAt(controls.target)

          morph = 0
          globeLayer.setMorph(0)
        } else {
          // 阶段 2（45% ~ 100% 时间）：球体已完全立正且正对视野，接缝 100% 位于正后背（-Z 轴）。
          // 此时相机与姿态静止，网格如画卷般从容展开，接缝从背面盲区自然滑向两侧，视野内全程纯净！
          earthGroup.quaternion.copy(quatTo)
          camera.position.copy(cameraTo)
          controls.target.set(0, 0, 0)
          camera.lookAt(controls.target)

          const meshProgress = (rawProgress - splitPoint) / (1 - splitPoint)
          const meshEased = easeMorph(meshProgress)
          morph = THREE.MathUtils.lerp(0, 1, meshEased)
          globeLayer.setMorph(morph)
        }
      } else {
        // 2D -> 3D 收纳管线（两阶段错峰动力学）：
        // 阶段 1（0 ~ 55% 时间）：网格优先收拢回完整球体（morph: 1 -> 0），相机与姿态保持平稳正立
        const splitPoint = 0.55
        if (rawProgress <= splitPoint) {
          earthGroup.quaternion.copy(quatFrom)
          const meshProgress = rawProgress / splitPoint
          const meshEased = easeMorph(meshProgress)
          controls.target.copy(targetFrom).multiplyScalar(1 - meshEased)
          camera.position.copy(cameraFrom).sub(targetFrom.clone().multiplyScalar(meshEased))
          camera.lookAt(controls.target)

          morph = THREE.MathUtils.lerp(1, 0, meshEased)
          globeLayer.setMorph(morph)
        } else {
          // 阶段 2（55% ~ 100% 时间）：球体已完全闭合密封（morph = 0）。
          // 此时从容将完整球体倾斜过渡到目标 3D 姿态，并恢复用户离开前的观察视角！
          morph = 0
          globeLayer.setMorph(0)

          const rotProgress = (rawProgress - splitPoint) / (1 - splitPoint)
          const rotEased = easeMorph(rotProgress)
          earthGroup.quaternion.slerpQuaternions(quatFrom, quatTo, rotEased)

          const targetDist = THREE.MathUtils.lerp(cameraFrom.length(), cameraTo.length(), rotEased)
          camera.position.lerpVectors(cameraFrom, cameraTo, rotEased).setLength(targetDist)
          controls.target.set(0, 0, 0)
          camera.lookAt(controls.target)
        }
      }

      if (rawProgress >= 1) finishMorph()
    }

    const animate = () => {
      if (disposed) return

      const elapsed = clock.getDelta()
      const delta = Math.min(0.05, elapsed)

      if (morphing) {
        advanceMorph(delta)
      } else if (projection === '3d') {
        // 1. Angular velocity damping on right-click release
        if (!isRightDragging && Math.abs(angularVelocity) > 0.01) {
          const stepAngle = angularVelocity * delta
          const deltaQuat = new THREE.Quaternion().setFromAxisAngle(angularVelocityAxis, stepAngle)
          earthGroup.quaternion.premultiply(deltaQuat)
          angularVelocity *= Math.pow(inertiaDamping, delta * 60)
          if (Math.abs(angularVelocity) <= 0.01) {
            angularVelocity = 0
          }
          realignTimer = 0
          isRealigning = false
          emitViewStateChange()
        } else if (!isRightDragging && angularVelocity !== 0) {
          angularVelocity = 0
        }

        // 2. Realignment & Auto-rotation state machine
        if (!isRightDragging && !isOrbitDragging && Math.abs(angularVelocity) <= 0.01) {
          const worldNorth = new THREE.Vector3(0, 1, 0).applyQuaternion(earthGroup.quaternion)
          const camDir = camera.position.clone().sub(controls.target).normalize()
          const isNorthUpright = worldNorth.y > 0.999
          const isCameraLevel = Math.abs(camDir.y) < 0.005
          const isUpright = isNorthUpright && isCameraLevel

          if (isUpright && !isRealigning) {
            realignTimer += delta
            if (realignTimer >= realignDelay && autoRotation) {
              const autoAngle = delta * 0.025
              const autoQuat = new THREE.Quaternion().setFromAxisAngle(new THREE.Vector3(0, 1, 0), autoAngle)
              earthGroup.quaternion.premultiply(autoQuat)
              emitViewStateChange()
            }
          } else {
            if (!isRealigning) {
              realignTimer += delta
              if (realignTimer >= realignDelay) {
                isRealigning = true
                realignProgress = 0
                realignStartQuat.copy(earthGroup.quaternion)
                realignStartCamPos.copy(camera.position)

                const currentDist = camera.position.distanceTo(controls.target)
                const invQuat = earthGroup.quaternion.clone().invert()
                const localCamDir = camDir.clone().applyQuaternion(invQuat)
                const localH = new THREE.Vector3(localCamDir.x, 0, localCamDir.z)
                if (localH.lengthSq() < 1e-6) localH.set(1, 0, 0)
                else localH.normalize()

                const camH = new THREE.Vector3(camDir.x, 0, camDir.z)
                if (camH.lengthSq() < 1e-6) camH.set(0, 0, 1)
                else camH.normalize()

                realignTargetCamPos.copy(camH).multiplyScalar(currentDist)

                const sinPhi = localH.z * camH.x - localH.x * camH.z
                const cosPhi = localH.x * camH.x + localH.z * camH.z
                const phi = Math.atan2(sinPhi, cosPhi)
                realignTargetQuat.setFromAxisAngle(new THREE.Vector3(0, 1, 0), phi)

                if (realignStartQuat.dot(realignTargetQuat) < 0) {
                  realignTargetQuat.x = -realignTargetQuat.x
                  realignTargetQuat.y = -realignTargetQuat.y
                  realignTargetQuat.z = -realignTargetQuat.z
                  realignTargetQuat.w = -realignTargetQuat.w
                }
              }
            } else {
              realignProgress = Math.min(1, realignProgress + delta / realignDuration)
              const t = realignProgress
              const easeT = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2

              earthGroup.quaternion.slerpQuaternions(realignStartQuat, realignTargetQuat, easeT)

              const currentDist = realignStartCamPos.distanceTo(controls.target)
              camera.position
                .lerpVectors(realignStartCamPos, realignTargetCamPos, easeT)
                .setLength(currentDist)
              camera.lookAt(controls.target)

              emitViewStateChange()
              if (realignProgress >= 1) {
                isRealigning = false
                realignTimer = 0
                earthGroup.quaternion.copy(realignTargetQuat)
                camera.position.copy(realignTargetCamPos)
                camera.lookAt(controls.target)
                controls.enableDamping = false
                controls.update()
                controls.enableDamping = true
              }
            }
          }
        } else {
          realignTimer = 0
          isRealigning = false
        }

        if (!isRealigning) {
          controls.update(delta)
        }
      } else {
        controls.update(delta)
      }

      endpointLayer.update(delta)
      if (visualMode === 'space' && morph === 0) globeLayer.syncSunLight()
      routeLayer.update(elapsed)

      render()
    }

    const updateSunForTime = () => {
      getRealtimeSunDirection(new Date(), sunDirection )
      globeLayer.setSunDirection(sunDirection)
      endpointLayer.setSunDirection(sunDirection)
    }

    const updateAnimationLoop = () => {
      if (disposed) return

      renderer.setAnimationLoop(null)
      clock.stop()

      // Without a running loop there is nothing to drive the morph, so settle it
      // immediately rather than leaving the overlays hidden mid-transition.
      if (morphing && (reducedMotion || !visible || !intersecting)) finishMorph()

      if (!visible || !intersecting) return

      if (visualMode === 'space' && morph === 0) updateSunForTime()

      if (reducedMotion) {
        controls.enableDamping = false
        render()
      } else {
        controls.enableDamping = true
        clock.start()
        renderer.setAnimationLoop(animate)
      }
    }

    const startMorph = (next: EarthProjection) => {
      if (isRightDragging) {
        isRightDragging = false
        activeRightPointerId = null
        controls.enabled = true
      }
      angularVelocity = 0
      isRealigning = false
      realignTimer = 0

      // Only a settled globe is worth remembering: reversing mid-transition must
      // keep the view the user actually left, not a half-unrolled camera.
      if (projection === '3d' && !morphing) {
        orbitCameraPosition.copy(camera.position)
        orbitQuat.copy(earthGroup.quaternion)
        orbitCenterLongitude = view.centerLongitude
      }

      if (next === '2d') {
        // 动态计算当前 3D 视角正对屏幕中心的经度 (Facing Longitude)
        const camDir = camera.position.clone().sub(controls.target).normalize()
        const invQuat = earthGroup.quaternion.clone().invert()
        const localCamDir = camDir.clone().applyQuaternion(invQuat)
        const horizontalDistSq = localCamDir.x * localCamDir.x + localCamDir.z * localCamDir.z
        const localAngle =
          horizontalDistSq > 1e-4
            ? Math.atan2(localCamDir.x, localCamDir.z) * (180 / Math.PI)
            : 0

        // 关键点：保持连续累加，禁止取模/wrapLongitude！
        // 保证 centerOffset 的偏移变化量严格等于 localAngle / 360，
        // 与旋转量 qOffset 严格精确抵消，彻底根除跨 ±180° 日界线时产生的 360° 突转！
        const facingLongitude = view.centerLongitude + localAngle

        view = { ...view, projection: next, centerLongitude: facingLongitude }
        globeLayer.setCenterLongitude(view.centerLongitude)
        applyView()

        // 抵消纹理经度偏移，使三维球体在视觉上绝对无突跳
        const qOffset = new THREE.Quaternion().setFromAxisAngle(
          new THREE.Vector3(0, 1, 0),
          THREE.MathUtils.degToRad(localAngle),
        )
        orbitQOffset.copy(qOffset)
        earthGroup.quaternion.multiply(qOffset)

        quatFrom.copy(earthGroup.quaternion)

        // 构造 2D 目标姿态：使展开后的 2D 平面直接正对当前视线的水平方向 (forward)
        const forward = new THREE.Vector3(camDir.x, 0, camDir.z)
        if (forward.lengthSq() < 1e-6) forward.set(0, 0, 1)
        else forward.normalize()

        const up = new THREE.Vector3(0, 1, 0)
        const right = new THREE.Vector3().crossVectors(up, forward).normalize()
        const rotMatrix = new THREE.Matrix4().makeBasis(right, up, forward)
        quatTo.setFromRotationMatrix(rotMatrix)

        // 保证四元数插值走正向半球（最短弧），杜绝反向或 360° 翻转
        if (quatFrom.dot(quatTo) < 0) {
          quatTo.x = -quatTo.x
          quatTo.y = -quatTo.y
          quatTo.z = -quatTo.z
          quatTo.w = -quatTo.w
        }

        cameraTo.copy(forward).multiplyScalar(mapViewDistance())
      } else {
        view = { ...view, projection: next }
        quatTo.copy(orbitQuat).multiply(orbitQOffset)
        cameraTo.copy(orbitCameraPosition)
        quatFrom.copy(earthGroup.quaternion)
        if (quatFrom.dot(quatTo) < 0) {
          quatTo.x = -quatTo.x
          quatTo.y = -quatTo.y
          quatTo.z = -quatTo.z
          quatTo.w = -quatTo.w
        }
      }

      projection = next
      morphFrom = morph
      morphTo = projectionMorph(next)
      morphElapsed = 0
      morphing = true
      cameraFrom.copy(camera.position)
      targetFrom.copy(controls.target)

      controls.enabled = false
      overlayGroup.visible = false
      updateAnimationLoop()
    }

    const showEndpoint = (event: PointerEvent, pin = false) => {
      const bounds = renderer.domElement.getBoundingClientRect()
      const endpoint = endpointLayer.hitTest(event.clientX, event.clientY, bounds)

      if (endpoint) {
        pinnedEndpoint = pin
        options.onEndpointHover(endpoint, event.clientX, event.clientY)
        renderer.domElement.style.cursor = 'pointer'
      } else if (!pinnedEndpoint || pin) {
        pinnedEndpoint = false
        options.onEndpointHover(null)
        renderer.domElement.style.cursor = ''
      }
    }
    const onContextMenu = (event: MouseEvent) => {
      event.preventDefault()
    }
    const onPointerDown = (event: PointerEvent) => {
      realignTimer = 0
      isRealigning = false

      if (event.pointerType === 'mouse' && event.button === 2 && projection === '3d' && !morphing) {
        isRightDragging = true
        activeRightPointerId = event.pointerId
        angularVelocity = 0
        lastTrackballVec.copy(getTrackballVector(event.clientX, event.clientY, renderer.domElement))
        renderer.domElement.setPointerCapture?.(event.pointerId)
        controls.enabled = false
      }
    }
    const onPointerMove = (event: PointerEvent) => {
      if (isRightDragging && projection === '3d' && event.pointerId === activeRightPointerId) {
        const currentVec = getTrackballVector(event.clientX, event.clientY, renderer.domElement)
        const dot = THREE.MathUtils.clamp(lastTrackballVec.dot(currentVec), -1, 1)
        const angle = Math.acos(dot) * 2.0

        if (angle > 1e-4) {
          const axis = new THREE.Vector3().crossVectors(lastTrackballVec, currentVec).normalize()
          const worldAxis = axis.clone().applyQuaternion(camera.quaternion)
          const deltaQuat = new THREE.Quaternion().setFromAxisAngle(worldAxis, angle)
          earthGroup.quaternion.premultiply(deltaQuat)

          angularVelocityAxis.copy(worldAxis)
          angularVelocity = angle * 30
          lastTrackballVec.copy(currentVec)
          emitViewStateChange()
        }
        return
      }

      if (!pinnedEndpoint && event.pointerType !== 'touch') showEndpoint(event)
    }
    const onPointerUp = (event: PointerEvent) => {
      isOrbitDragging = false
      if (isRightDragging && event.pointerId === activeRightPointerId) {
        isRightDragging = false
        activeRightPointerId = null
        if (renderer.domElement.hasPointerCapture?.(event.pointerId)) {
          renderer.domElement.releasePointerCapture(event.pointerId)
        }
        controls.enabled = !morphing
      }
    }
    const onClick = (event: PointerEvent) => showEndpoint(event, true)
    const onPointerLeave = () => {
      if (!pinnedEndpoint) options.onEndpointHover(null)
    }
    const onWheel = () => {
      realignTimer = 0
      isRealigning = false
      emitViewStateChange()
    }

    renderer.domElement.addEventListener('contextmenu', onContextMenu)
    renderer.domElement.addEventListener('pointerdown', onPointerDown)
    renderer.domElement.addEventListener('pointermove', onPointerMove)
    renderer.domElement.addEventListener('pointerup', onPointerUp)
    renderer.domElement.addEventListener('pointercancel', onPointerUp)
    renderer.domElement.addEventListener('click', onClick)
    renderer.domElement.addEventListener('pointerleave', onPointerLeave)
    renderer.domElement.addEventListener('wheel', onWheel, { passive: true })
    registerCleanup(() => {
      renderer.domElement.removeEventListener('contextmenu', onContextMenu)
      renderer.domElement.removeEventListener('pointerdown', onPointerDown)
      renderer.domElement.removeEventListener('pointermove', onPointerMove)
      renderer.domElement.removeEventListener('pointerup', onPointerUp)
      renderer.domElement.removeEventListener('pointercancel', onPointerUp)
      renderer.domElement.removeEventListener('click', onClick)
      renderer.domElement.removeEventListener('pointerleave', onPointerLeave)
      renderer.domElement.removeEventListener('wheel', onWheel)
    })

    const resizeObserver = new ResizeObserver(([entry]) => {
      const width = Math.max(1, entry.contentRect.width)
      const height = Math.max(1, entry.contentRect.height)

      // How far the settled map is zoomed in relative to its fitted distance, so
      // the viewport can change without the camera drifting to another zoom.
      const zoomRatio =
        projection === '2d' && !morphing
          ? camera.position.distanceTo(controls.target) / mapDistance()
          : 0

      camera.aspect = width / height
      camera.updateProjectionMatrix()
      renderer.setSize(width, height, false)
      labelRenderer.setSize(width, height)

      // The fitted map distance depends on the aspect ratio, so the zoom range —
      // and the pending morph destination — have to follow the viewport.
      if (projection === '2d') {
        applyControls()
        if (morphing) {
          const dir = cameraTo.clone().sub(controls.target).normalize()
          if (dir.lengthSq() < 1e-6) dir.set(0, 0, 1)
          cameraTo.copy(controls.target).addScaledVector(dir, mapViewDistance())
        } else {
          const dir = camera.position.clone().sub(controls.target).normalize()
          if (dir.lengthSq() < 1e-6) dir.set(0, 0, 1)
          camera.position.copy(controls.target).addScaledVector(dir, mapDistance() * zoomRatio)
          controls.update()
        }
      }

      render()
    })
    resizeObserver.observe(container)
    registerCleanup(() => resizeObserver.disconnect())

    const intersectionObserver = new IntersectionObserver(
      ([entry]) => {
        intersecting = entry.isIntersecting
        updateAnimationLoop()
      },
      { threshold: 0.01 },
    )
    intersectionObserver.observe(container)
    registerCleanup(() => intersectionObserver.disconnect())

    const onVisibilityChange = () => {
      visible = !document.hidden
      updateAnimationLoop()
    }
    document.addEventListener('visibilitychange', onVisibilityChange)
    registerCleanup(() => document.removeEventListener('visibilitychange', onVisibilityChange))

    const sunTimer = window.setInterval(() => {
      if (visualMode === 'space' && morph === 0) updateSunForTime()
      if (reducedMotion) render()
    }, 60_000)
    registerCleanup(() => window.clearInterval(sunTimer))

    const onControlsChange = () => {
      if (reducedMotion) render()
    }
    controls.addEventListener('change', onControlsChange)
    registerCleanup(() => controls.removeEventListener('change', onControlsChange))

    globeLayer.setMorph(morph)
    applyControls()
    if (projection === '2d') {
      camera.position.set(0, 0, mapViewDistance())
      controls.update()
    }
    updateAnimationLoop()

    return {
      setRoutes(incomingRoutes) {
        if (disposed) return

        const snapshot = createEarthRenderSnapshot(incomingRoutes)
        const topologyChanged = snapshot.signature !== currentSignature
        currentSignature = snapshot.signature
        routeLayer.setSnapshot(snapshot, topologyChanged)
        const endpoints = endpointLayer.setSnapshot(snapshot, topologyChanged)
        cityLabelLayer.setEndpoints(endpoints)
        if (reducedMotion) render()
      },
      setInitialLocation(location) {
        if (
          disposed ||
          initialLocationSet ||
          !Number.isFinite(location.latitude) ||
          !Number.isFinite(location.longitude)
        ) {
          return
        }

        initialLocationSet = true
        // Centring the world on the user applies to both projections: it is what
        // puts them in the middle of the flat map, and it moves the seam to their
        // antipode so their busiest routes are not the ones cut in half.
        view = { ...view, centerLongitude: location.longitude }
        globeLayer.setCenterLongitude(view.centerLongitude)
        updateSunForTime()
        applyView()

        // Aiming the camera, on the other hand, only means anything on a globe.
        if (projection === '2d' || morphing) pendingInitialLocation = location
        else applyInitialLocation(location)

        render()
      },
      setReducedMotion(value) {
        if (disposed) return
        reducedMotion = value
        updateAnimationLoop()
      },
      setAutoRotation(enabled) {
        if (disposed) return
        autoRotation = enabled
      },
      setCityLabelsVisible(nextVisible) {
        if (disposed) return
        cityLabelLayer.setVisible(nextVisible)
        render()
      },
      setProjection(next) {
        if (disposed || projection === next) return
        startMorph(next)
      },
      setVisualMode(mode) {
        if (disposed || visualMode === mode) return
        visualMode = mode
        globeLayer.setVisualMode(mode)
        routeLayer.setVisualMode(mode)
        endpointLayer.setVisualMode(mode)
        render()
      },
      setColorScheme(scheme) {
        if (disposed || colorScheme === scheme) return
        colorScheme = scheme
        globeLayer.setColorScheme(scheme)
        routeLayer.setColorScheme(scheme)
        if (visualMode === 'flat' || visualMode === 'color' || projection === '2d') render()
      },
      getViewState: () => ({
        quaternion: [
          earthGroup.quaternion.x,
          earthGroup.quaternion.y,
          earthGroup.quaternion.z,
          earthGroup.quaternion.w,
        ],
        distance: camera.position.distanceTo(controls.target),
      }),
      setViewState: (state) => {
        if (disposed) return
        if (Array.isArray(state.quaternion) && state.quaternion.length === 4) {
          earthGroup.quaternion.fromArray(state.quaternion)
        }
        if (typeof state.distance === 'number') {
          const dir = camera.position.clone().sub(controls.target).normalize()
          camera.position.copy(controls.target).addScaledVector(dir, state.distance)
          controls.update()
        }
        render()
      },
      dispose: disposeResources,
    }
  } catch (error) {
    try {
      disposeResources()
    } catch {
      // Preserve the initialization error while still attempting every cleanup.
    }
    throw error
  }
}
