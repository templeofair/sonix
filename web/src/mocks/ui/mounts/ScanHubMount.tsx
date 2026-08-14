import Card from '../../../shared/components/Card'
import { Link } from 'react-router-dom'
import { buttonVariantClass } from '../../../shared/components/Button'

/** Scan hub stand-in matching product Add chooser shape. */
export default function ScanHubMount() {
  return (
    <div className="space-y-4 max-w-md">
      <h2 className="text-lg font-semibold text-gray-900 tracking-tight">Scan letters</h2>
      <div className="grid gap-3">
        <Card className="p-5 shadow-card">
          <p className="text-2xl mb-2" aria-hidden>
            📷
          </p>
          <p className="font-medium text-gray-900">Camera</p>
          <p className="text-sm text-muted mt-1 mb-3">Capture pages with the device camera.</p>
          <Link to="/__ui/camera" className={buttonVariantClass('primary') + ' inline-flex px-4 py-2'}>
            Open camera placeholder
          </Link>
        </Card>
        <Card className="p-5 shadow-card">
          <p className="text-2xl mb-2" aria-hidden>
            📁
          </p>
          <p className="font-medium text-gray-900">Upload</p>
          <p className="text-sm text-muted mt-1">Images or PDF from files.</p>
        </Card>
      </div>
    </div>
  )
}
