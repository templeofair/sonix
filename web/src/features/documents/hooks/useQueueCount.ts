import { useCallback, useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { documentsApi } from '../services/documentsApi'

const QUEUE_STATUS = 'pending,processing,failed,partial'

/**
 * Shared queue total for the My letters tab badge (sidebar / mobile).
 * One list call with comma-separated statuses (supported by the documents API).
 */
export function useQueueCount(refreshKey = 0) {
  const location = useLocation()
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [failed, setFailed] = useState(false)

  const reload = useCallback(() => {
    let cancelled = false
    setLoading(true)
    documentsApi
      .list({ status: QUEUE_STATUS, limit: 1 })
      .then((res) => {
        if (!cancelled) {
          setTotal(res.total)
          setFailed(false)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setTotal(0)
          setFailed(true)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    const cancel = reload()
    return cancel
  }, [reload, location.pathname, refreshKey])

  return { total, loading, failed, reload }
}
