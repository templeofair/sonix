# Sonix

Germany is good at many things. Short letters are not one of them.

Sonix is a hobby project for that doormat pile: scan, organise, and translate letters and documents on your own machine. Also a journey with AI tools — the model reads the Amtsdeutsch; you keep the coffee. Self-hosted; nothing goes to a Sonix cloud.

You get searchable text (Tesseract OCR or Ollama vision), an English translation, a short summary, and a document date — then search, tag, and export.

**Privacy:** page images and text **are sent to the Ollama server you configure**. If that is the same host, nothing leaves the machine. If you point Settings at another host, your documents go there.

## Requirements

- Docker with Compose
- An [Ollama](https://ollama.com) host with a **vision-capable** model pulled (a few GB). Compose defaults to `gemma3:latest`.
- A trusted LAN. This is a **single-account** app, not a multi-user internet service.

## Install

```bash
git clone https://github.com/templeofair/sonix.git
cd sonix
cp .env.example .env
```

Edit `.env`:

1. `SESSION_SECRET` — `openssl rand -hex 32` (required; the app will not start without it)
2. `SEED_USERNAME` / `SEED_PASSWORD` — first login only (password at least 12 characters)
3. `OLLAMA_BASE_URL` — if Ollama is not on the Docker host

```bash
docker compose up -d --build
```

Open `http://<host>:9080` and sign in. For the phone camera, use `https://<host>:9443` and accept the self-signed certificate once.

After the first login, remove `SEED_PASSWORD` from `.env` and recreate the container.

Compose v1 (`docker-compose`) works; v2 (`docker compose`) is preferred.

## What it does

1. Capture (phone camera), upload images/PDF, or drop files into an inbox folder.
2. Extract: per-page vision **or** Tesseract, then translation, summary, and date.
3. Browse **My letters** / **Explore**, search, tag, export a zip from Settings.

There is no in-app user registration.

## Configure

All variables: [`.env.example`](.env.example) and [`docs/configuration.md`](docs/configuration.md).

Useful knobs: `OLLAMA_VISION_MODEL` / `OLLAMA_TEXT_MODEL`, `EXTRACTION_MAX_CONCURRENT` (default 1), `DOCUMENT_MAX_PAGES` (default 50). Models and the Ollama URL are also editable in **Settings**.

SQLite and files live in the `sonix-data` volume (`DATA_DIR`). There is **no migrate command** — the schema is created on first start. Back up that volume to back up everything.

HTTPS: a self-signed cert is minted under `DATA_DIR/tls` (valid one year). `./scripts/update-container.sh` can add the host LAN IP to the certificate SAN.

Optional HP Scan-to-Computer sidecar: [`docs/auto-import-scans.md`](docs/auto-import-scans.md).

## Limitations

- **Single tenant:** every session sees every document.
- **Trusted LAN only.** Self-signed TLS; no service worker; install-to-home-screen is manifest-only.
- **No registration, no password-change UI, no offline mode.**
- Extraction needs a reachable Ollama for the default vision path. The library still starts if Ollama is down.
- Login rate limit: 8 attempts per IP per minute.

## Docs

- [Architecture](docs/architecture.md) · [Deploy](docs/deployment.md) · [API](docs/api.md) · [Troubleshooting](docs/troubleshooting.md)
- [Extraction](docs/extraction.md) · [Configuration](docs/configuration.md) · [Security](SECURITY.md)
- [Contributing](CONTRIBUTING.md) · [Third-party notices](THIRD-PARTY-NOTICES.md)

Licence: [MIT](LICENSE).
