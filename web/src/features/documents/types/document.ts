/** Document domain types shared by the documents feature (API shapes). */

export type DocumentListSort = 'created_desc' | 'date_desc' | 'date_asc'

export type DocumentListItem = {
  id: number
  title?: string
  status: string
  created_at: string
  updated_at: string
  document_date?: string
  page_count: number
  thumbnail_available: boolean
}

export type DocumentDetail = {
  id: number
  title?: string
  status: string
  created_at: string
  updated_at: string
  extraction_error?: string
  /** Set while processing: sequential page pipeline progress */
  extraction_pages_done?: number
  extraction_pages_total?: number
  pages: { page_index: number; content_type: string }[]
  page_count: number
  thumbnail_available: boolean
  extraction?: {
    tags: string[]
    summary: string
    document_date?: string
    extracted_at: string
    engine_id?: string
    prompt_version?: string
    /** Wall-clock ms for the last successful extraction job (server-side). */
    extraction_wall_ms?: number
    /** Present only when GET uses `?include=text`. */
    full_text_original?: string
    full_text_english?: string
  }
}

export type DocumentListParams = {
  q?: string
  tag?: string
  /** Comma-separated upload years (OR). */
  year?: string
  document_date_from?: string
  document_date_to?: string
  created_from?: string
  created_to?: string
  status?: string
  /** `1` restricts the list to letters with no letter date (Explore → No date). */
  undated?: 1
  sort?: DocumentListSort
  page?: number
  limit?: number
}

export type DocumentListResponse = {
  documents: DocumentListItem[]
  total: number
}

/** One Explore folder: letters bucketed by the year of their own date. */
export type DocumentDateYear = {
  year: string
  count: number
}

export type DocumentDateYearsResponse = {
  years: DocumentDateYear[]
  undated_count: number
}

export type DocumentGetOptions = {
  /** When true, extraction includes full_text_original / full_text_english. */
  include?: 'text'
}
