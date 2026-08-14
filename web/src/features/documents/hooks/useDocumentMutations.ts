import { useCallback, useState, type Dispatch, type SetStateAction } from 'react'
import type { NavigateFunction } from 'react-router-dom'
import { documentsApi } from '../services/documentsApi'
import type { DocumentDetail } from '../types/document'
import { useAppNav } from '../../../lib/appNav'

/** Document detail mutations: delete, tags, date, title, extraction. */
export function useDocumentMutations(
  doc: DocumentDetail | null,
  setDoc: Dispatch<SetStateAction<DocumentDetail | null>>,
  refresh: () => void,
  navigate: NavigateFunction
) {
  const { appPath } = useAppNav()
  const [extracting, setExtracting] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [savingTags, setSavingTags] = useState(false)
  const [savingDate, setSavingDate] = useState(false)
  const [savingTitle, setSavingTitle] = useState(false)

  const deleteDocument = () => {
    if (!doc) return
    if (!confirm('Delete this document? This cannot be undone.')) return
    setDeleting(true)
    documentsApi.delete(doc.id).then(() => navigate(appPath('/'))).finally(() => setDeleting(false))
  }

  const putTags = (next: string[]) => {
    if (!doc) return Promise.resolve()
    setSavingTags(true)
    return documentsApi
      .putTags(doc.id, next)
      .then(() => {
        setDoc((prev) => {
          if (!prev) return prev
          const base = prev.extraction ?? { tags: [], summary: '', extracted_at: '' }
          return { ...prev, extraction: { ...base, tags: next } }
        })
      })
      .finally(() => setSavingTags(false))
  }

  const putDocumentDate = (value: string | null) => {
    if (!doc) return Promise.resolve()
    setSavingDate(true)
    return documentsApi
      .putDocumentDate(doc.id, value)
      .then(() => {
        setDoc((prev) => {
          if (!prev?.extraction) return prev
          return { ...prev, extraction: { ...prev.extraction, document_date: value ?? undefined } }
        })
      })
      .finally(() => setSavingDate(false))
  }

  const putTitle = (value: string) => {
    if (!doc) return Promise.resolve()
    setSavingTitle(true)
    return documentsApi
      .putTitle(doc.id, value)
      .then(() => {
        setDoc((prev) => (prev ? { ...prev, title: value || undefined } : prev))
      })
      .finally(() => setSavingTitle(false))
  }

  const startExtract = (useOcr?: boolean) => {
    if (!doc) return Promise.resolve()
    setExtracting(true)
    return documentsApi
      .extract(doc.id, useOcr !== undefined ? { use_ocr: useOcr } : undefined)
      .then(() => {
        // Stay on the detail UI immediately; refresh may briefly fail under DB load.
        setDoc((prev) => (prev ? { ...prev, status: 'processing', extraction_error: undefined } : prev))
        refresh()
      })
      .finally(() => setExtracting(false))
  }

  const resetExtraction = () => {
    if (!doc) return
    setDoc((prev) => (prev ? { ...prev, status: 'pending', extraction_error: undefined } : prev))
    documentsApi.resetExtraction(doc.id).then(() => refresh()).catch(() => refresh())
  }

  const [imageRev, setImageRev] = useState(0)

  const pageImageUrl = useCallback(
    (pageIndex: number) => {
      if (!doc) return ''
      const url = documentsApi.pageImageUrl(doc.id, pageIndex)
      if (url.startsWith('data:')) return url
      return `${url}?v=${imageRev}`
    },
    [doc, imageRev]
  )

  const rotatePage = (pageIndex: number, degrees: 90 | 180 | 270) => {
    if (!doc) return Promise.resolve()
    return documentsApi.rotatePage(doc.id, pageIndex, degrees).then(() => {
      setImageRev((n) => n + 1)
    })
  }

  return {
    extracting,
    deleting,
    savingTags,
    savingDate,
    savingTitle,
    deleteDocument,
    putTags,
    putDocumentDate,
    putTitle,
    startExtract,
    resetExtraction,
    pageImageUrl,
    rotatePage,
  }
}
