import type { DocumentListSort } from '../types/document'
import type { LibraryLayout } from '../lib/libraryParams'
import { STATUS_FILTER_CHIPS } from '../lib/libraryParams'
import MultiCheckSelect from '../../../shared/components/MultiCheckSelect'

type Props = {
  layout: LibraryLayout
  sort: DocumentListSort
  statusValues: string[]
  tagValues: string[]
  yearValues: string[]
  tagOptions: string[]
  yearOptions: string[]
  onLayoutChange: (layout: LibraryLayout) => void
  onSortChange: (sort: DocumentListSort) => void
  onStatusChange: (values: string[]) => void
  onTagChange: (values: string[]) => void
  onYearChange: (values: string[]) => void
}

const statusOptions = STATUS_FILTER_CHIPS.filter((c) => c.value !== '').map((c) => ({
  value: c.value,
  label: c.label,
}))

/** Layout / sort / multi-select filters for the flat library. */
export default function LibraryToolbar({
  layout,
  sort,
  statusValues,
  tagValues,
  yearValues,
  tagOptions,
  yearOptions,
  onLayoutChange,
  onSortChange,
  onStatusChange,
  onTagChange,
  onYearChange,
}: Props) {
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="inline-flex rounded-btn border border-border overflow-hidden" role="group" aria-label="Layout">
          <button
            type="button"
            aria-pressed={layout === 'grid'}
            onClick={() => onLayoutChange('grid')}
            className={`control min-h-[44px] px-3 text-sm font-medium ${
              layout === 'grid' ? 'bg-accent text-white' : 'bg-card text-gray-800 hover:bg-gray-50'
            }`}
          >
            Grid
          </button>
          <button
            type="button"
            aria-pressed={layout === 'list'}
            onClick={() => onLayoutChange('list')}
            className={`control min-h-[44px] px-3 text-sm font-medium border-l border-border ${
              layout === 'list' ? 'bg-accent text-white' : 'bg-card text-gray-800 hover:bg-gray-50'
            }`}
          >
            List
          </button>
        </div>
        <label className="sr-only" htmlFor="library-sort">
          Sort
        </label>
        <select
          id="library-sort"
          value={sort}
          onChange={(e) => onSortChange(e.target.value as DocumentListSort)}
          className="control min-h-[44px] px-3 border border-border rounded-btn bg-card text-sm text-gray-900"
        >
          <option value="created_desc">Newest upload</option>
          <option value="date_desc">Document date ↓</option>
          <option value="date_asc">Document date ↑</option>
        </select>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <MultiCheckSelect
          id="library-filter-status"
          label="Status"
          emptyLabel="Any status"
          options={statusOptions}
          values={statusValues}
          onChange={onStatusChange}
        />
        <MultiCheckSelect
          id="library-filter-tags"
          label="Tags"
          emptyLabel="Any tag"
          options={tagOptions.map((t) => ({ value: t, label: t }))}
          values={tagValues}
          onChange={onTagChange}
        />
        <MultiCheckSelect
          id="library-filter-year"
          label="Year"
          emptyLabel="Any year"
          options={yearOptions.map((y) => ({ value: y, label: y }))}
          values={yearValues}
          onChange={onYearChange}
        />
      </div>
    </div>
  )
}

/** Phone is always a single column; the grid option is an `md+` affordance. */
export function documentListClassName(layout: LibraryLayout): string {
  return layout === 'grid' ? 'grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-3' : 'space-y-3'
}
