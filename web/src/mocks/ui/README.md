# UI mock playground (`/__ui`)

**Full-app Sonix mirror** for UX work (DEV only). Same pages and chrome as the product; **fake data**; actions succeed in the UI but **never** touch the real server or database.

## Open

Vite DEV → `http://localhost:<port>/__ui`

- Home = real My letters (fixture letters)
- Same tabs: My letters / Explore / Scan / Settings
- Explore folder index, year folders and `no-date`, document detail, add/upload/review
- Camera: real shell; no hardware capture
- Secondary **component kit**: `/__ui/_kit`

Banner on mock screens: “UI mock — fake data only…”

## What is shared vs not

| Shared with product | Not shared |
|---------------------|------------|
| `Layout`, feature pages, shared components, tokens | Letter **content** (fixtures vs SQLite) |
| Editing those files while viewing `/__ui` | Real `/api`, Ollama, disk |

`http://127.0.0.1:5173/documents/45` is the **product**. Changing that letter does **not** change mock fixtures.

## Safe actions

Delete, rename, tags, extract, settings save, export, upload: UI runs to completion; mock API updates **in-memory** fixtures only. Refresh `/__ui` resets to the seed set.

## Process: RESET → WORK → ACCEPT / REJECT

| Step | Meaning |
|------|---------|
| **RESET** | Clear `experiments/*`. Full app shows current product UI + seed fixtures. |
| **WORK** | Prefer editing product files while viewing `/__ui` (already shared), or short-lived `experiments/<slug>/`. |
| **ACCEPT** | If experiments: promote into product; then RESET. If you edited product files: run gate. |
| **REJECT** | RESET experiments / revert product edits. |

## Rules

- Product code must **never** import from `web/src/mocks/`.
- Mock installs API handler only while MockApp is mounted (`installApiMock`).
- Not a fourth primary tab in the product IA.
