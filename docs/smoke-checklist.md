# Manual smoke checklist (Sonix web)

Run after UI changes or before a release. Requires a working backend (local or Docker), seed or test user, and optional Ollama for extraction-heavy steps.

## Automated gate (run first)

```bash
./scripts/regression-gate.sh
# Backend-only: SKIP_WEB_BUILD=1 ./scripts/regression-gate.sh
```

Or see [regression-gates.md](regression-gates.md) for tiers and alternatives.

## Auth

| Step | Action | Expect |
|------|--------|--------|
| A1 | Open `/login` | Card layout, fields labeled, Sign in disabled until filled |
| A2 | Wrong password | Inline error in bordered message |
| A3 | Valid login | Redirect to `/` (My letters) |

## Shell (desktop + narrow viewport)

| Step | Action | Expect |
|------|--------|--------|
| S1 | Desktop sidebar: My letters, Explore, Scan letters, Settings | Order matches mobile; active route highlighted; keyboard Tab shows visible focus ring on links |
| S2 | Mobile width: no hamburger / no left drawer for primary nav | Top strip shows Sonix brand + section title on the right |
| S3 | Bottom nav (mobile): My letters, Explore, **Scan**, Settings | Matches routes; Scan uses FAB-style control (label **Scan**, not “Scan letters”); My letters shows a queue count badge when `pending/processing/failed/partial` &gt; 0 (accessible name includes the count) |
| S3b | Mobile ~360px: four bottom tabs | Labels readable without truncation; safe-area padding intact |
| S4 | Settings → Export data | Triggers download (auth cookie) |
| S5 | Settings → Log out | Returns to login (visible at all widths; desktop also keeps header Log out) |
| S6 | Stop the Sonix server, reload the app | Full-page **Cannot reach Sonix** with Try again (not an empty list / not a blank login) |
| S7 | Android Chrome → Install app / Add to Home screen | Standalone launch, Sonix icon, accent theme colour, no browser address bar |

## Browse & lists

| Step | Action | Expect |
|------|--------|--------|
| B1 | My letters default (Recent 15) | Shows ≤15 recent letters; **no Load more** on default view; slim overview + sticky search; **Select** for multi-delete; mobile title in top strip |
| B1b | My letters: older letter | Search finds a letter older than the Recent 15 set |
| B2 | Explore folder index | Years + **No date** (when undated exist); open a year sorted by letter date |
| B2b | Explore year + redirect | Back from year → `/explore`; `/year/2024` redirects to `/explore/2024` |
| B3 | Filters + Search | `q` / document dates / multi status·tag·year round-trip in URL; More options comboboxes; Load more when filtered/searched; `/search?q=…` → library |
| B4 | Queue badge + Status filter | My letters nav badge shows queue count; recreate queue via **More options → Status** (pending+failed+partial); `/pending` still lands on that status set |
| B5 | Library cards | Grid/list thumbs share a 3:4 frame; short tap opens document; press-and-hold thumb opens preview (Close / Escape / backdrop); **Preview first page** control also opens preview |
| B6 | Explore on phone | Folder tiles usable; Explore tab works on mobile |

## Add flows

| Step | Action | Expect |
|------|--------|--------|
| C1 | Add hub | Two CTAs; back link to My letters |
| C2 | Upload: images | Choose images → `/add/review`; reorder/rotate/delete; Save uploads with per-page progress |
| C2b | Upload: PDF | PDF list + Upload creates doc and navigates to detail (direct path) |
| C3 | Camera (HTTPS/local only) | Black editor shell; Cancel/Done (no Back); draft kept when returning from review; capture → Done → review |
| C3b | Review | Move ←/→, rotate; icon **Crop** / **Colour** / Add / Retake / Delete + **Save**; Cancel → “Discard this scan?” (Keep editing / Discard); Colour on stills; delete last → camera |
| C3c | Crop editor | Drag corners; Reset / Full page; Done applies perspective; Cancel leaves page unchanged |
| C3d | Colour modes | On review only; Original unchanged; Clean improves readability; prefs remembered |
| C4 | Document detail → Extraction mode → OCR (Tesseract) → extract | New runs store `engine_id` like `tesseract:deu+eng`; German text should be markedly better than pre-Phase-1 English-only OCR |

## Document detail

| Step | Action | Expect |
|------|--------|--------|
| D1 | Title edit | Tap name → edit; blur/Enter with change → Save name change? dialog; Cancel discards; Escape discards; phone uses icon Back/Delete |
| D2 | Tags / date | Persists |
| D3 | Translation / original text | View translation / View original text open reader overlay (full-screen on phone); Close / Escape dismisses. Copy works |
| D4 | Re-process / reset | Per status; action button inside the related box (not sticky); OCR checkbox; failed/partial show a brief error (no raw dump) |
| D5 | Pages viewer | Vertical thumb rail; Full screen / Share / **Rotate clockwise**; indicator “N of M”; rotate updates image + thumb |
| D6 | Engine meta | Visible on phone for OCR and AI runs |

## Accessibility / contrast (spot check)

- Tab through primary page: every interactive control shows a **visible focus** state.
- Body text: primary copy uses `text-gray-800` / `text-gray-900`; secondary uses `text-muted` — verify readability on `bg-surface` / `bg-card` in sunlight or high brightness.
- Images in app: page thumbnails and scans have non-empty `alt` where applicable.

## Auto-import scans

| Step | Action | Expect |
|------|--------|--------|
| H1 | Settings → enable **auto-import** → Save → copy a PDF/JPEG into `$DATA_DIR/inbox` | File moves to `inbox/processed/`; new letter in My letters within ~few seconds |
| H2 | Disable auto-import → Save → drop another file | File stays in inbox (not consumed) |
| H3 | Settings → set **Printer IP** → Save | Value stored; `$DATA_DIR/hp-scan/printer_ip` updated |
| H3b | **Save** Printer IP, then **Test printer** | Success banner if that saved IP is reachable; fail if offline. Unsaved typed IP is ignored (button disabled until Save) |
| H3c | Change **Printer IP** in Settings → Save (helper running) | Within ~10s helper logs `printer IP changed, restarting` then `using printer IP=<new>`; walk-up scan still works |
| H3d | `HP_SCAN_IP_WATCH=0` on hp-scan | Restores manual recreate behaviour (no auto-restart on IP change) |
| H4 | (Optional) HP helper: panel **Sonix** scan | Same as H1 without manual copy — see [auto-import-scans.md](auto-import-scans.md) |

## Record outcome

Note date, commit hash, and any failures in the PR or release notes.
