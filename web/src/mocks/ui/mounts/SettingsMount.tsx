import Card from '../../../shared/components/Card'
import SectionLabel from '../../../shared/components/SectionLabel'
import Field from '../../../shared/components/Field'
import Button from '../../../shared/components/Button'

/** Settings-shaped fixture (real Field/Card/Button). Full SettingsPage needs useSettings API. */
export default function SettingsMount() {
  return (
    <div className="space-y-6 max-w-lg">
      <Card className="p-4 sm:p-5 space-y-4 shadow-card">
        <SectionLabel as="h2">Ollama</SectionLabel>
        <Field id="mock-ollama-url" label="URL" defaultValue="http://127.0.0.1:11434" />
        <Field id="mock-ollama-model" label="Model" defaultValue="llama3.2-vision" />
        <div className="flex gap-2">
          <Button type="button">Save</Button>
          <Button type="button" variant="secondary">
            Test connection
          </Button>
        </div>
      </Card>
      <Card className="p-4 sm:p-5 space-y-3 shadow-card">
        <SectionLabel as="h2">Account</SectionLabel>
        <Button type="button" variant="secondary" className="w-full">
          Export data
        </Button>
        <Button type="button" variant="danger" className="w-full">
          Log out
        </Button>
      </Card>
      <p className="text-xs text-muted-subtle">Fixture form — not wired to `/api/settings`.</p>
    </div>
  )
}
