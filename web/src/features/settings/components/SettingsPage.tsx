import type { ReactNode } from 'react'
import { useAuth } from '../../auth'
import { exportUrl } from '../services/settingsApi'
import { useSettings } from '../hooks/useSettings'
import PageHeader from '../../../components/PageHeader'
import Banner from '../../../shared/components/Banner'
import Field from '../../../shared/components/Field'
import Card from '../../../shared/components/Card'
import Skeleton from '../../../shared/components/Skeleton'
import ExtractModeSelect from '../../../shared/components/ExtractModeSelect'

/** Matches Account actions: full-width bordered controls, equal halves in a row. */
const actionBtn =
  'control flex-1 min-w-0 min-h-11 px-3 py-2.5 rounded-btn text-sm font-medium border border-border bg-white text-center transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none'
const actionBtnPrimary = `${actionBtn} text-gray-800 hover:bg-surface`
const actionBtnSecondary = `${actionBtn} text-muted hover:bg-surface hover:text-gray-800`
const accountBtn =
  'block w-full min-h-11 text-center px-4 py-2.5 rounded-btn text-sm font-medium border border-border bg-white transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-2'

function ActionRow({ children }: { children: ReactNode }) {
  return <div className="flex gap-2 w-full pt-1">{children}</div>
}

/** Progressive reveal: expands/collapses height with opacity (respects reduced-motion via index.css). */
function ExpandReveal({ open, children }: { open: boolean; children: ReactNode }) {
  return (
    <div
      className={`grid transition-[grid-template-rows] duration-300 ease-out motion-reduce:transition-none ${
        open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'
      }`}
      aria-hidden={!open}
    >
      <div className="min-h-0 overflow-hidden">
        <div
          className={`transition-opacity duration-300 ease-out motion-reduce:transition-none ${
            open ? 'opacity-100' : 'opacity-0'
          }`}
        >
          {children}
        </div>
      </div>
    </div>
  )
}

export default function SettingsPage() {
  const { logout } = useAuth()
  const {
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
  } = useSettings()

  const exportRaw = exportUrl()
  const exportHref =
    exportRaw.startsWith('data:') || exportRaw.startsWith('blob:')
      ? exportRaw
      : typeof window !== 'undefined'
        ? `${window.location.origin}${exportRaw}`
        : exportRaw
  const busy = saving || testing || testingPrinter
  const printerDirty = hpPrinterIP.trim() !== savedPrinterIP.trim()
  const canTestPrinter = Boolean(savedPrinterIP.trim()) && !printerDirty

  const onImportEnabledChange = (checked: boolean) => {
    setImportEnabled(checked)
    if (!checked) {
      setImportAutoExtract(false)
      setImportExtractUseOcr(false)
    }
  }

  const onImportAutoExtractChange = (checked: boolean) => {
    setImportAutoExtract(checked)
    if (!checked) {
      setImportExtractUseOcr(false)
    }
  }

  return (
    <>
      <PageHeader title="Settings" subtitle="Ollama, extraction, and auto-import scans" />
      <div className="flex-1 overflow-y-auto">
        <div className="p-4 sm:p-6 max-w-3xl mx-auto pb-24 md:pb-8 w-full">
          {loading ? (
            <Skeleton className="h-40 w-full" />
          ) : (
            <>
              <Card className="p-5 sm:p-6 space-y-6">
                <h2 className="text-sm font-semibold text-gray-900">Ollama</h2>
                <Field
                  id="ollama"
                  label="Ollama server"
                  type="text"
                  value={ollamaUrl}
                  onChange={(e) => setOllamaUrl(e.target.value)}
                  placeholder="e.g. host.docker.internal or 10.0.0.1:11434"
                >
                  <p className="text-xs text-muted mt-2 leading-relaxed">
                    Host and optional port of the machine running Ollama (default port{' '}
                    <span className="tabular-nums">11434</span>).
                  </p>
                </Field>
                <Field
                  id="ollama-model"
                  label="Model for scanning pages"
                  type="text"
                  value={ollamaModel}
                  onChange={(e) => setOllamaModel(e.target.value)}
                  placeholder="e.g. Keyvan/german-ocr-turbo"
                >
                  <p className="text-xs text-muted mt-2 leading-relaxed">
                    Vision model that reads page images. For German letters: Keyvan/german-ocr-turbo.
                  </p>
                </Field>
                <Field
                  id="ollama-text-model"
                  label="Model for translation and summary"
                  type="text"
                  value={ollamaTextModel}
                  onChange={(e) => setOllamaTextModel(e.target.value)}
                  placeholder="e.g. Keyvan/german-text-3.1"
                >
                  <p className="text-xs text-muted mt-2 leading-relaxed">
                    Text model for English translation, summary, and document date. Leave empty to reuse the scanning
                    model. Recommended: Keyvan/german-text-3.1.
                  </p>
                </Field>
                <ActionRow>
                  <button
                    type="button"
                    className={actionBtnSecondary}
                    onClick={testConnection}
                    disabled={busy}
                  >
                    {testing ? 'Testing…' : 'Test connection'}
                  </button>
                  <button type="button" className={actionBtnPrimary} onClick={save} disabled={busy}>
                    {saving ? 'Saving…' : 'Save settings'}
                  </button>
                </ActionRow>
              </Card>

              <Card className="mt-8 p-5 sm:p-6 space-y-4">
                <h2 className="text-sm font-semibold text-gray-900">Auto-import scans</h2>
                <p className="text-xs text-muted leading-relaxed">
                  When enabled, PDF or image files in the scan folder become letters (from the scanner helper or a
                  manual copy).
                </p>
                <Field
                  id="hp-printer-ip"
                  label="Printer IP"
                  type="text"
                  inputMode="decimal"
                  autoComplete="off"
                  value={hpPrinterIP}
                  onChange={(e) => setHpPrinterIP(e.target.value)}
                  placeholder="192.168.1.50"
                >
                  <p className="text-xs text-muted mt-2 leading-relaxed">
                    Wi‑Fi address of the OfficeJet. Save, then Test printer. A changed IP applies within about 10 seconds.
                  </p>
                </Field>
                <div className="space-y-1">
                  <label className="flex items-start gap-3 min-h-11 cursor-pointer">
                    <input
                      id="import-enabled"
                      type="checkbox"
                      className="mt-1 h-4 w-4 accent-[var(--color-accent)]"
                      checked={importEnabled}
                      onChange={(e) => onImportEnabledChange(e.target.checked)}
                    />
                    <span className="text-sm text-gray-900">Enable auto-import</span>
                  </label>
                  <ExpandReveal open={importEnabled}>
                    <label className="flex items-start gap-3 min-h-11 cursor-pointer pt-1">
                      <input
                        id="import-auto-extract"
                        type="checkbox"
                        className="mt-1 h-4 w-4 accent-[var(--color-accent)]"
                        checked={importAutoExtract}
                        tabIndex={importEnabled ? 0 : -1}
                        onChange={(e) => onImportAutoExtractChange(e.target.checked)}
                      />
                      <span className="text-sm text-gray-900">Extract after import</span>
                    </label>
                  </ExpandReveal>
                  <ExpandReveal open={importEnabled && importAutoExtract}>
                    <div className="pt-1 pl-7">
                      <ExtractModeSelect
                        id="import-extract-mode"
                        value={importExtractUseOcr ? 'ocr' : 'llm'}
                        onChange={(mode) => setImportExtractUseOcr(mode === 'ocr')}
                      />
                    </div>
                  </ExpandReveal>
                </div>
                <ActionRow>
                  <button
                    type="button"
                    className={actionBtnSecondary}
                    onClick={testPrinter}
                    disabled={busy || !canTestPrinter}
                  >
                    {testingPrinter ? 'Testing…' : 'Test printer'}
                  </button>
                  <button type="button" className={actionBtnPrimary} onClick={save} disabled={busy}>
                    {saving ? 'Saving…' : 'Save settings'}
                  </button>
                </ActionRow>
              </Card>

              {feedback && (
                <div className="mt-6" aria-live="polite">
                  <Banner tone={feedback.tone}>{feedback.text}</Banner>
                </div>
              )}
            </>
          )}

          <Card className="mt-8 p-5 sm:p-6 space-y-3">
            <h2 className="text-sm font-semibold text-gray-900">Account</h2>
            <a href={exportHref} className={`${accountBtn} text-gray-800 hover:bg-surface`}>
              Export data
            </a>
            <button
              type="button"
              onClick={() => logout()}
              className={`${accountBtn} text-muted hover:bg-surface hover:text-gray-800`}
            >
              Log out
            </button>
          </Card>
        </div>
      </div>
    </>
  )
}
