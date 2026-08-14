import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import {
  newDraftPageId,
  reorderList,
  revokePageUrl,
  type DraftPage,
} from '../lib/captureDraft'

type CaptureDraftValue = {
  pages: DraftPage[]
  title: string
  setTitle: (title: string) => void
  /** Replace page at this index on next capture; null = append. */
  retakeIndex: number | null
  setRetakeIndex: (index: number | null) => void
  addBlob: (blob: Blob, source?: DraftPage['source'], name?: string) => void
  addFiles: (files: File[]) => Promise<number>
  replaceAt: (index: number, blob: Blob, source?: DraftPage['source']) => void
  /** Replace every page blob in order (preserves source/name; resets rotation). */
  replaceAll: (blobs: Blob[]) => void
  removeAt: (index: number) => void
  move: (from: number, to: number) => void
  moveLeft: (index: number) => void
  moveRight: (index: number) => void
  rotateAt: (index: number) => void
  clear: () => void
}

const CaptureDraftContext = createContext<CaptureDraftValue | null>(null)

function pageFromBlob(blob: Blob, source: DraftPage['source'], name?: string): DraftPage {
  return {
    id: newDraftPageId(),
    blob,
    url: URL.createObjectURL(blob),
    rotation: 0,
    source,
    name,
  }
}

export function CaptureDraftProvider({ children }: { children: ReactNode }) {
  const [pages, setPages] = useState<DraftPage[]>([])
  const [title, setTitle] = useState('')
  const [retakeIndex, setRetakeIndex] = useState<number | null>(null)

  const clear = useCallback(() => {
    setPages((prev) => {
      prev.forEach(revokePageUrl)
      return []
    })
    setTitle('')
    setRetakeIndex(null)
  }, [])

  const addBlob = useCallback((blob: Blob, source: DraftPage['source'] = 'camera', name?: string) => {
    setPages((prev) => [...prev, pageFromBlob(blob, source, name)])
  }, [])

  const addFiles = useCallback(async (files: File[]) => {
    const images = files.filter((f) => f.type.startsWith('image/'))
    if (images.length === 0) return 0
    const created = await Promise.all(
      images.map(async (f) => {
        const buf = await f.arrayBuffer()
        const blob = new Blob([buf], { type: f.type || 'image/jpeg' })
        return pageFromBlob(blob, 'upload', f.name)
      })
    )
    setPages((prev) => [...prev, ...created])
    return created.length
  }, [])

  const replaceAt = useCallback((index: number, blob: Blob, source: DraftPage['source'] = 'camera') => {
    setPages((prev) => {
      if (index < 0 || index >= prev.length) {
        return [...prev, pageFromBlob(blob, source)]
      }
      const next = prev.slice()
      revokePageUrl(next[index])
      next[index] = pageFromBlob(blob, source, next[index].name)
      return next
    })
    setRetakeIndex(null)
  }, [])

  const replaceAll = useCallback((blobs: Blob[]) => {
    setPages((prev) => {
      prev.forEach(revokePageUrl)
      return blobs.map((blob, i) =>
        pageFromBlob(blob, prev[i]?.source ?? 'camera', prev[i]?.name)
      )
    })
    setRetakeIndex(null)
  }, [])

  const removeAt = useCallback((index: number) => {
    setPages((prev) => {
      if (index < 0 || index >= prev.length) return prev
      const next = prev.slice()
      const [removed] = next.splice(index, 1)
      revokePageUrl(removed)
      return next
    })
    setRetakeIndex((ri) => {
      if (ri == null) return null
      if (ri === index) return null
      if (ri > index) return ri - 1
      return ri
    })
  }, [])

  const move = useCallback((from: number, to: number) => {
    setPages((prev) => reorderList(prev, from, to))
  }, [])

  const moveLeft = useCallback((index: number) => {
    if (index <= 0) return
    setPages((prev) => reorderList(prev, index, index - 1))
  }, [])

  const moveRight = useCallback((index: number) => {
    setPages((prev) => {
      if (index < 0 || index >= prev.length - 1) return prev
      return reorderList(prev, index, index + 1)
    })
  }, [])

  const rotateAt = useCallback((index: number) => {
    setPages((prev) => {
      if (index < 0 || index >= prev.length) return prev
      const next = prev.slice()
      const page = next[index]
      const rotation = ((page.rotation + 90) % 360) as DraftPage['rotation']
      next[index] = { ...page, rotation }
      return next
    })
  }, [])

  const value = useMemo(
    () => ({
      pages,
      title,
      setTitle,
      retakeIndex,
      setRetakeIndex,
      addBlob,
      addFiles,
      replaceAt,
      replaceAll,
      removeAt,
      move,
      moveLeft,
      moveRight,
      rotateAt,
      clear,
    }),
    [
      pages,
      title,
      retakeIndex,
      addBlob,
      addFiles,
      replaceAt,
      replaceAll,
      removeAt,
      move,
      moveLeft,
      moveRight,
      rotateAt,
      clear,
    ]
  )

  return <CaptureDraftContext.Provider value={value}>{children}</CaptureDraftContext.Provider>
}

export function useCaptureDraft(): CaptureDraftValue {
  const ctx = useContext(CaptureDraftContext)
  if (!ctx) throw new Error('useCaptureDraft requires CaptureDraftProvider')
  return ctx
}
