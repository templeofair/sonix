import { useCallback, useRef, useState } from 'react'
import type { RefObject } from 'react'

/** Ideal camera constraints for document capture (4K; devices may negotiate down). */
export const CAMERA_VIDEO_IDEAL = {
  facingMode: { ideal: 'environment' as const },
  width: { ideal: 3840 },
  height: { ideal: 2160 },
}

type TrackCaps = MediaTrackCapabilities & {
  torch?: boolean
  focusMode?: string[]
  pointsOfInterest?: boolean
}

function readCaps(track: MediaStreamTrack | undefined): TrackCaps | null {
  if (!track || typeof track.getCapabilities !== 'function') return null
  try {
    return track.getCapabilities() as TrackCaps
  } catch {
    return null
  }
}

export function useCameraCapture() {
  const [streaming, setStreaming] = useState(false)
  const [videoReady, setVideoReady] = useState(false)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [flash, setFlash] = useState(false)
  const [torchSupported, setTorchSupported] = useState(false)
  const [torchOn, setTorchOn] = useState(false)
  const [focusSupported, setFocusSupported] = useState(false)
  const videoRef = useRef<HTMLVideoElement>(null)
  const streamRef = useRef<MediaStream | null>(null)

  const refreshCapabilities = useCallback((stream: MediaStream) => {
    const track = stream.getVideoTracks()[0]
    const caps = readCaps(track)
    setTorchSupported(Boolean(caps?.torch))
    const modes = caps?.focusMode ?? []
    setFocusSupported(
      Boolean(caps?.pointsOfInterest) &&
        (modes.includes('manual') || modes.includes('single-shot') || modes.includes('continuous'))
    )
    setTorchOn(false)
  }, [])

  const startCamera = useCallback(async () => {
    setCameraError(null)
    setVideoReady(false)
    setTorchSupported(false)
    setFocusSupported(false)
    setTorchOn(false)

    const isSecure =
      location.protocol === 'https:' || location.hostname === 'localhost' || location.hostname === '127.0.0.1'
    if (!navigator.mediaDevices?.getUserMedia) {
      if (!isSecure) {
        const httpsUrl = `https://${location.hostname}:9443${location.pathname}`
        setCameraError(`Camera requires HTTPS. Open this page via HTTPS: ${httpsUrl}`)
      } else {
        setCameraError('Camera is not supported in this browser. Try Safari, Chrome, or Firefox.')
      }
      return
    }
    try {
      let stream: MediaStream | null = null
      try {
        // Prefer 4K so A4 lettering yields ~30px capital height for Tesseract (~300 DPI).
        // `ideal` negotiates down on devices that cannot honour 3840x2160.
        stream = await navigator.mediaDevices.getUserMedia({
          video: CAMERA_VIDEO_IDEAL,
          audio: false,
        })
      } catch {
        stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false })
      }
      streamRef.current = stream
      refreshCapabilities(stream)
      if (videoRef.current) {
        videoRef.current.srcObject = stream
        videoRef.current.onloadedmetadata = () => {
          setVideoReady(true)
        }
        try {
          await videoRef.current.play()
        } catch {
          /* autoplay attr handles it */
        }
      }
      setStreaming(true)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Camera access denied or not available'
      if (msg.includes('denied') || msg.includes('NotAllowed')) {
        setCameraError('Camera permission denied. Please allow camera access in your browser settings and try again.')
      } else {
        setCameraError(msg)
      }
    }
  }, [refreshCapabilities])

  const stopCamera = useCallback(() => {
    streamRef.current?.getTracks().forEach((t) => t.stop())
    streamRef.current = null
    if (videoRef.current) {
      videoRef.current.srcObject = null
      videoRef.current.onloadedmetadata = null
    }
    setStreaming(false)
    setVideoReady(false)
    setTorchSupported(false)
    setFocusSupported(false)
    setTorchOn(false)
  }, [])

  const setTorch = useCallback(async (on: boolean) => {
    const track = streamRef.current?.getVideoTracks()[0]
    if (!track) return
    const caps = readCaps(track)
    if (!caps?.torch) return
    try {
      await track.applyConstraints({
        advanced: [{ torch: on } as MediaTrackConstraintSet],
      })
      setTorchOn(on)
    } catch {
      /* capability reported but apply failed — leave UI state unchanged */
    }
  }, [])

  /** Normalised viewfinder coords (0–1). No-op when focus is unsupported. */
  const tapToFocus = useCallback(async (nx: number, ny: number) => {
    const track = streamRef.current?.getVideoTracks()[0]
    if (!track) return
    const caps = readCaps(track)
    const modes = caps?.focusMode ?? []
    if (!caps?.pointsOfInterest) return
    const x = Math.min(1, Math.max(0, nx))
    const y = Math.min(1, Math.max(0, ny))
    const focusMode = modes.includes('manual')
      ? 'manual'
      : modes.includes('single-shot')
        ? 'single-shot'
        : modes.includes('continuous')
          ? 'continuous'
          : null
    if (!focusMode) return
    try {
      await track.applyConstraints({
        advanced: [
          {
            focusMode,
            pointsOfInterest: [{ x, y }],
          } as MediaTrackConstraintSet,
        ],
      })
    } catch {
      /* ignore */
    }
  }, [])

  /** Capture current frame as a JPEG blob (caller owns the blob; no data-URL copy). */
  const capture = useCallback((): Promise<Blob | null> => {
    const video = videoRef.current
    if (!video || video.videoWidth === 0 || video.videoHeight === 0) {
      return Promise.resolve(null)
    }

    setFlash(true)
    setTimeout(() => setFlash(false), 150)

    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) return Promise.resolve(null)
    ctx.drawImage(video, 0, 0)

    return new Promise((resolve) => {
      canvas.toBlob(
        (blob) => {
          // Release canvas backing store promptly on mobile.
          canvas.width = 0
          canvas.height = 0
          resolve(blob)
        },
        'image/jpeg',
        0.92
      )
    })
  }, [])

  return {
    streaming,
    videoReady,
    cameraError,
    flash,
    torchSupported,
    torchOn,
    focusSupported,
    videoRef: videoRef as RefObject<HTMLVideoElement>,
    streamRef,
    startCamera,
    stopCamera,
    capture,
    setTorch,
    tapToFocus,
  }
}
