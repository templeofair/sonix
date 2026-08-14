import { useEffect, useRef, useState } from 'react'
import { settingsApi } from '../services/settingsApi'

export type SettingsFeedback = {
  tone: 'success' | 'error'
  text: string
}

const FEEDBACK_MS = 5000

export function useSettings() {
  const [ollamaUrl, setOllamaUrl] = useState('')
  const [ollamaModel, setOllamaModel] = useState('')
  const [ollamaTextModel, setOllamaTextModel] = useState('')
  const [importEnabled, setImportEnabled] = useState(false)
  const [importAutoExtract, setImportAutoExtract] = useState(true)
  const [importExtractUseOcr, setImportExtractUseOcr] = useState(true)
  const [hpPrinterIP, setHpPrinterIP] = useState('')
  const [savedPrinterIP, setSavedPrinterIP] = useState('')
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState(false)
  const [testingPrinter, setTestingPrinter] = useState(false)
  const [feedback, setFeedback] = useState<SettingsFeedback | null>(null)
  const feedbackTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const showFeedback = (next: SettingsFeedback) => {
    if (feedbackTimer.current) {
      clearTimeout(feedbackTimer.current)
      feedbackTimer.current = null
    }
    setFeedback(next)
    feedbackTimer.current = setTimeout(() => {
      setFeedback(null)
      feedbackTimer.current = null
    }, FEEDBACK_MS)
  }

  const clearFeedback = () => {
    if (feedbackTimer.current) {
      clearTimeout(feedbackTimer.current)
      feedbackTimer.current = null
    }
    setFeedback(null)
  }

  useEffect(() => {
    return () => {
      if (feedbackTimer.current) clearTimeout(feedbackTimer.current)
    }
  }, [])

  useEffect(() => {
    settingsApi
      .get()
      .then((d) => {
        setOllamaUrl(d.ollama_base_url_raw ?? '')
        setOllamaModel(d.ollama_model_raw ?? d.ollama_model ?? '')
        setOllamaTextModel(d.ollama_text_model_raw ?? d.ollama_text_model ?? '')
        const enabled = !!d.import_inbox_enabled
        const autoExtract = enabled && d.import_auto_extract !== false
        setImportEnabled(enabled)
        setImportAutoExtract(autoExtract)
        // OCR only applies when extract-after is on; never show a stale checked state.
        setImportExtractUseOcr(autoExtract && d.import_extract_use_ocr !== false)
        const ip = d.hp_printer_ip ?? ''
        setHpPrinterIP(ip)
        setSavedPrinterIP(ip)
      })
      .catch(() => {
        setOllamaUrl('')
        setOllamaModel('')
        setOllamaTextModel('')
      })
      .finally(() => setLoading(false))
  }, [])

  const save = async () => {
    clearFeedback()
    setSaving(true)
    const ipBefore = savedPrinterIP
    const extractOn = importEnabled && importAutoExtract
    const useOcr = extractOn && importExtractUseOcr
    try {
      const res = await settingsApi.put({
        ollama_base_url: ollamaUrl.trim(),
        ollama_model: ollamaModel.trim(),
        ollama_text_model: ollamaTextModel.trim(),
        import_inbox_enabled: importEnabled,
        import_auto_extract: extractOn,
        import_extract_use_ocr: useOcr,
        hp_printer_ip: hpPrinterIP.trim(),
      })
      const nextEnabled = !!res.import_inbox_enabled
      const nextExtract = nextEnabled && res.import_auto_extract !== false
      setImportEnabled(nextEnabled)
      setImportAutoExtract(nextExtract)
      setImportExtractUseOcr(nextExtract && res.import_extract_use_ocr === true)
      const nextIP = res.hp_printer_ip ?? hpPrinterIP.trim()
      setHpPrinterIP(nextIP)
      setSavedPrinterIP(nextIP)
      const ipChanged = nextIP !== ipBefore
      showFeedback({
        tone: 'success',
        text: ipChanged
          ? 'Settings saved. The scan helper picks up the new printer IP within about 10 seconds.'
          : 'Settings saved.',
      })
    } catch (err) {
      showFeedback({
        tone: 'error',
        text: err instanceof Error && err.message ? err.message : 'Failed to save.',
      })
    } finally {
      setSaving(false)
    }
  }

  const testConnection = async () => {
    clearFeedback()
    setTesting(true)
    try {
      const res = await settingsApi.testOllama()
      if (res.ok) {
        showFeedback({ tone: 'success', text: 'Ollama is reachable. Configured models are available.' })
      } else {
        showFeedback({
          tone: 'error',
          text: res.error || 'Ollama connection failed. Check the URL and that Ollama is running.',
        })
      }
    } catch {
      showFeedback({
        tone: 'error',
        text: 'Ollama connection failed. Check the URL and that Ollama is running.',
      })
    } finally {
      setTesting(false)
    }
  }

  const testPrinter = async () => {
    clearFeedback()
    setTestingPrinter(true)
    try {
      const res = await settingsApi.testPrinter()
      if (res.ok) {
        showFeedback({ tone: 'success', text: 'Printer is reachable on the network.' })
      } else {
        showFeedback({
          tone: 'error',
          text: res.error || 'Could not reach the printer. Check the IP and Wi‑Fi.',
        })
      }
    } catch {
      showFeedback({
        tone: 'error',
        text: 'Could not reach the printer. Check the IP and Wi‑Fi.',
      })
    } finally {
      setTestingPrinter(false)
    }
  }

  return {
    ollamaUrl,
    setOllamaUrl,
    ollamaModel,
    setOllamaModel,
    ollamaTextModel,
    setOllamaTextModel,
    importEnabled,
    setImportEnabled,
    importAutoExtract,
    setImportAutoExtract,
    importExtractUseOcr,
    setImportExtractUseOcr,
    hpPrinterIP,
    setHpPrinterIP,
    saving,
    loading,
    testing,
    testingPrinter,
    feedback,
    save,
    testConnection,
    testPrinter,
    savedPrinterIP,
  }
}
