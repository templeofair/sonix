# Auto-import scans (scan folder + HP OfficeJet 250)

**Who this is for:** Operators wiring a scanner (especially HP OfficeJet 250) so finished scans become Sonix letters without using the phone upload UI.

**In one sentence:** Drop a finished PDF/JPEG into `DATA_DIR/inbox` (or let the HP helper do it); Sonix creates one letter per file.

**UI name:** Settings → **Auto-import scans**.

---

## Concern split

| Piece | Job |
|-------|-----|
| **Sonix** | Watch `DATA_DIR/inbox`; one stable file → one document; optional extract. No duplex logic. |
| **HP helper** (`node-hp-scan-to`) | Talk to the printer; two panel targets (**Sonix**, **Sonix duplex**); merge duplex; write **finished** PDFs into the inbox. Page JPEGs stay under `TEMP_DIR` (`/tmp/hp-scan`) while assembling. |

---

## Sonix scan folder (inbox)

| Path | Role |
|------|------|
| `$DATA_DIR/inbox/` | Drop zone (PDF, JPEG, PNG, WebP) |
| `$DATA_DIR/inbox/processed/` | Successfully imported (moved out of the drop zone) |
| `$DATA_DIR/inbox/failed/` | Import errors |

**Settings → Auto-import scans**

- **Printer IP** — OfficeJet Wi‑Fi address; written to `$DATA_DIR/hp-scan/printer_ip` for the helper. **Save**, then **Test printer** (the test uses the saved IP only).
- **Enable auto-import** — when off, files are left untouched; extract options are hidden and cleared.
- **Extract after import** — appears when auto-import is on. Turning it off hides and clears the OCR option.
- **Extraction mode** — appears when extract-after is on: choose **OCR (Tesseract)** or **LLM vision**. This is also the suggested default on document Extract / Re-process when a letter has no prior extraction mode.

### Changing Printer IP (applies automatically)

Save a new **Printer IP** and the helper picks it up **within about 10 seconds** — no terminal command. The helper entrypoint watches the `hp-scan/` directory (`inotifyd`, plus a 3 s reconcile tick as insurance) and on a valid, different IP sends `SIGTERM` to s6 (PID 1); the container exits cleanly and `restart: unless-stopped` brings it back reading the new file. Logs show `hp-scan: printer IP changed, restarting` then `hp-scan: using printer IP=<new>`.

- **Finish or cancel an in-flight duplex job first.** The restart drops a duplex job that is waiting for even pages — the captured odds are lost. Complete the second pass, or flush by selecting **Sonix**, before changing the IP.
- Ignored: the same value rewritten, and invalid or partially written content. Restarts are rate-limited to one per 30 s.
- Clearing the IP restarts into the existing “no printer IP yet” state.
- **Kill switch:** `HP_SCAN_IP_WATCH=0` in the helper environment disables the watcher (log line: `hp-scan: IP watcher disabled`). Then a new IP needs the manual recreate:

```bash
# Compose V2 (recommended when available):
docker compose -f docker-compose.yml -f deploy/hp-scan/compose.hp-scan.yml --profile hp-scan up -d --force-recreate
# Compose 1.29 fallback (full data volume mount):
docker-compose --profile hp-scan up -d --force-recreate
```

If the watcher itself dies, the helper keeps scanning on its current IP and logs `hp-scan: IP watcher stopped` once — the same manual recreate applies a new IP.

(Helper entrypoint prefers the file from Settings over `HP_SCAN_IP`.)

Host-side check of the watcher logic (no Docker, no printer): `./scripts/test-hp-ip-watch.sh`.

**Env (defaults)** — API/settings keys stay `import_inbox_*` for stability.

| Variable | Default | Notes |
|----------|---------|-------|
| `IMPORT_INBOX_DIR` | `$DATA_DIR/inbox` | Watch folder |
| `IMPORT_INBOX_ENABLED` | `false` | Env default; Settings override once saved |
| `IMPORT_AUTO_EXTRACT` | `true` | Settings override once saved |

### Manual test (no printer)

1. Start Sonix; open **Settings → Auto-import scans** → enable → Save.
2. Copy a PDF or JPEG into the inbox (Compose: `docker compose exec app ls /app/data/inbox`, or copy into the `sonix-data` volume’s `inbox/` folder).
3. Within a few seconds the file moves to `processed/` and a letter appears under **My letters** (and Queue if extract runs).

---

## HP OfficeJet 250 (helper on Sonix host)

OJ250 has no native “scan to NAS folder.” Use Scan-to-Computer via the helper on the same machine as Sonix.

```bash
# Prefer: set Printer IP in Sonix Settings → Auto-import scans, then:
# Compose V2 + Docker Engine 26+ (subpath mounts — helper sees inbox + hp-scan only):
docker compose -f docker-compose.yml -f deploy/hp-scan/compose.hp-scan.yml --profile hp-scan up -d
# Compose 1.29 (docker-compose): full sonix-data volume — see § Security
docker-compose --profile hp-scan up -d
# Optional env fallback: export HP_SCAN_IP=192.168.x.x
# Optional panel name (default LABEL is Sonix): HP_SCAN_LABEL=Sonix in .env
# Health HTTP port (host network; default 3001 — avoid EADDRINUSE if host :3000 is taken):
#   HP_SCAN_HEALTH_PORT=3001
```

The helper always starts an HTTP `/health` endpoint (`--health-check`). Pinned **1.8.0** `/app.sh` has no env for the port, so Compose passes `--health-check-port` via `CMDLINE` (default **3001**, override with `HP_SCAN_HEALTH_PORT` in `.env`). Stock default was **3000**, which collides with anything else bound on the host when `network_mode: host`. NODE_ENV / node-config warnings in helper logs are harmless.

After code changes, `./scripts/update-container.sh` rebuilds **app** and brings **hp-scan** back when any of: `--hp`, `WITH_HP_SCAN=1`, `HP_SCAN_LABEL` / `HP_SCAN_IP` in the environment or `.env`, or an existing `hp-scan` container. Hosts without HP leave the profile off (no third-party helper). The script prefers `docker compose` (v2) and falls back to `docker-compose` (v1; profiles since 1.28); prefer recreate-only-`hp-scan` if full recreate hits `KeyError: ContainerConfig` (see § Verify below). If another service already uses host port **3000**, set **`HP_SCAN_HEALTH_PORT`** (default **3001**) so the helper health check does not collide. Host-network for the **app** is a separate issue from hp-scan.

Panel targets (listen mode):

| Target | Use |
|--------|-----|
| **Sonix** | Single-sided letter (1…N pages) → one PDF |
| **Sonix duplex** | Pass 1 = odd pages (fronts); flip stack; pass 2 = even pages (backs) → one merged PDF |

**Duplex destination wiring (pinned helper):** Compose sets `CMDLINE=--add-emulated-duplex --health-check-port …` because the digest-pinned `node-hp-scan-to` **1.8.0** image’s `/app.sh` does **not** translate `ADD_EMULATED_DUPLEX` into that flag (newer upstream does), and likewise has no env for `--health-check-port`. Default duplex label is `{LABEL} duplex` → **Sonix duplex**. After recreate, helper logs should show both destinations registered (e.g. `New Destination registered: Sonix duplex` or both names under `Host destinations fetched`).

**Duplex back-pass PDF intent (pinned 1.8.0):** Stock `listenCmd` left `scanToPdf=false` on the emulated-duplex **back** pass, so page JPEGs were written into `DIR` (`/scan/inbox`). Sonix (correctly dumb: one file → one letter) imported each JPEG; the helper then failed merge with `ENOENT`. Sonix mounts [`deploy/hp-scan/listenCmd.js`](../deploy/hp-scan/listenCmd.js) over `/app/commands/listenCmd.js` so backs inherit `scanToPdf` from the front pass (same fix as upstream master). The same overlay also corrects flush-on-**Sonix** (stock checked the *new* target’s duplex flag, so simplex never flushed pending odds). That file is a **MIT-licensed derivative** of Emmanuel Counasse’s [node-hp-scan-to](https://github.com/manuc66/node-hp-scan-to) 1.8.0 — copyright and licence: [`THIRD-PARTY-NOTICES.md`](../THIRD-PARTY-NOTICES.md). Drop that volume when the image digest includes those fixes. See [`deploy/hp-scan/README.md`](../deploy/hp-scan/README.md).

### Duplex contract (operator)

1. First duplex pass = odds (fronts), order preserved.  
2. Flip the stack carefully for **document-wise** assembly.  
3. Second duplex pass = evens (backs).  
4. Finish within a reasonable time; prefer completing the second pass promptly.

### Duplex safety (helper)

| Situation | Expected |
|-----------|----------|
| Full duplex (odds + evens) | One merged PDF → inbox |
| Pending odds, then select **Sonix** | Helper flushes odds as one PDF, resets duplex, then runs the simplex job (proven 2026-07-30 with `listenCmd.js` overlay) |
| Pending odds, idle **10 minutes** | **Desired** flush as odds-only PDF. Stock `node-hp-scan-to` does **not** implement idle timeout yet; Compose sets `EMULATED_DUPLEX_IDLE_TIMEOUT_SEC=600` for a future helper that honors it. Until then: cancel pending duplex by selecting **Sonix**. |

Do not use `adf-autoscan` as the default — it conflicts with emulated duplex.

Third-party disclaimer: `node-hp-scan-to` is not affiliated with HP.

---

## Security

| Control | How |
|---------|-----|
| **Pinned images** | Sonix `Dockerfile` and Compose `hp-scan` use **digest pins** (not floating `:latest`). Bump digests only after inspecting the new image. |
| **Helper privileges** | Default: **no** `no-new-privileges` — on some Docker/kernel hosts it fails with `exec /bin/sh: operation not permitted` before the entrypoint runs. Optional: uncomment `security_opt` in Compose on hosts that tolerate it. Do not use `cap_drop: ALL` (upstream s6 needs `chown` on `/app`). Temp under `tmpfs`. |
| **Printer IP** | Settings validates **IPv4 only**; helper entrypoint re-validates the shared `printer_ip` file. |
| **Inbox import** | Rejects symlinks / non-regular files; max **50 MB** (same as HTTP upload). Auto-import defaults **off**. |
| **Trust boundary** | Enabling the `hp-scan` profile runs a third-party container. Treat it as trusted LAN software; keep auto-import off until you need it. |
| **Volume scope** | **Compose V2** (plugin ≥ 2.26, Engine ≥ 26): use [`deploy/hp-scan/compose.hp-scan.yml`](../deploy/hp-scan/compose.hp-scan.yml) so the helper mounts only `inbox/` and `hp-scan/` (subpath). Start the Sonix **app** first so those dirs exist in `sonix-data`. **Compose 1.29** cannot subpath-mount; stock [`docker-compose.yml`](../docker-compose.yml) mounts the full data volume — upgrade or accept that residual risk. |
| **Host network** | `network_mode: host` is often required for HP Walkup listen; it reduces network isolation. Revisit later if bridge + explicit IP works on your LAN. |

---

## Phase 0 spike checklist (printer)

- [x] Simplex **Sonix** → multi-page PDF in inbox → Sonix letter (doc #34, 2026-07-29)  
- [x] **Sonix duplex** appears on OJ250 Walk-up list (and in `docker-compose logs hp-scan`) — destinations registered 2026-07-30  
- [x] Duplex odds → flip → evens → **one** merged PDF in inbox → **one** letter (operator 2026-07-30 after `listenCmd.js` overlay; page sequence OK)  
- [x] Odds only, then select **Sonix** → odds flushed, then new simplex works (operator 2026-07-30)  
- [x] Note idle-timeout: stock 1.8.0 holds pending odds until flush via second duplex pass or **Sonix** (no 10‑min idle flush yet)

### Verify one letter = one finished PDF (operator)

1. Recreate **only** `hp-scan` (Compose 1.29 — avoid full recreate KeyError):

```bash
docker-compose --profile hp-scan stop hp-scan
docker-compose --profile hp-scan rm -f hp-scan
docker-compose --profile hp-scan up -d --no-deps hp-scan
```

2. Check logs for both targets + temp/inbox split:

```bash
docker-compose --profile hp-scan logs --tail=80 hp-scan
```

Expect: **Sonix** and **Sonix duplex** registered; `Target folder: /scan/inbox`; `Temp folder: /tmp/hp-scan`.

3. On the OJ250 panel: Scan → Computer (Walk-up) → both **Sonix** and **Sonix duplex**.

4. **Duplex smoke (one double-sided sheet or small stack):**
   - Select **Sonix duplex** → scan fronts (odds). Helper log: `saving front sides` and `using /tmp/hp-scan as temp…`. **No** new file in inbox yet; **no** new My letters row yet.
   - Flip stack → select **Sonix duplex** again → scan backs. Helper log: `saving back sides`, pages under `/tmp/hp-scan` (not `/scan/inbox`), then assembly → **one** `.pdf` in inbox.
   - Sonix moves that PDF to `inbox/processed/` and creates **exactly one** letter (page count = fronts + backs).

5. **Failure signatures (pre-overlay / misconfig):** backs logged as `contentType":"Photo"` and `Page downloaded to: /scan/inbox/….jpg`, then `ENOENT` — each JPEG becomes its own letter. That is helper writing intermediates into the drop zone, not Sonix inventing duplex merge.

Prefer **Extract after import** off while validating import-only behaviour.

---

## Cross-links

- Configuration: [configuration.md](configuration.md)  
- Compose: [`docker-compose.yml`](../docker-compose.yml) profile `hp-scan`; safer override [`deploy/hp-scan/compose.hp-scan.yml`](../deploy/hp-scan/compose.hp-scan.yml)  
- Host-network overlay (app): [`deploy/host-network/docker-compose.host.yml`](../deploy/host-network/docker-compose.host.yml)
