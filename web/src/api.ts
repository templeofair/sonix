/** Shared HTTP helpers. Document API lives in features/documents; settings in features/settings. */
export {
  documentsApi,
  type DocumentListItem,
  type DocumentDetail,
} from './features/documents'

export { settingsApi, exportUrl, type Settings } from './features/settings'
