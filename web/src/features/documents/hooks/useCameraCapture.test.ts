import { describe, it, expect } from 'vitest'
import { CAMERA_VIDEO_IDEAL } from './useCameraCapture'

describe('CAMERA_VIDEO_IDEAL', () => {
  it('requests 4K ideal resolution for OCR-friendly capital height', () => {
    expect(CAMERA_VIDEO_IDEAL.width.ideal).toBe(3840)
    expect(CAMERA_VIDEO_IDEAL.height.ideal).toBe(2160)
    expect(CAMERA_VIDEO_IDEAL.facingMode.ideal).toBe('environment')
  })
})
