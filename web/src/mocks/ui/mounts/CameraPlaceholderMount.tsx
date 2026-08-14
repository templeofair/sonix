import EmptyState from '../../../shared/components/EmptyState'

/** Device-only in product — gallery placeholder. */
export default function CameraPlaceholderMount() {
  return (
    <EmptyState title="Camera (device-only)">
      <p className="text-sm text-muted-subtle max-w-sm mx-auto">
        Real capture needs HTTPS + getUserMedia. Explore camera UX in the product app on a device.
      </p>
    </EmptyState>
  )
}
