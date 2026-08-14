# Deployment

## Compose

From a clone with a filled `.env`:

```bash
docker compose up -d --build
```

The app publishes **9080** (HTTP) and **9443** (HTTPS). `restart: unless-stopped` brings the container back when Docker starts. Enable Docker on boot if you want that after a host reboot.

There is **no database migrate command**. On first start, `internal/database` creates SQLite from `schema.sql` and applies small in-code migrations. The file appears under `DATA_DIR` (Compose: `/app/data` in volume `sonix-data`).

**Backup:** copy the volume / `DATA_DIR`. That is the whole library (DB, uploads, thumbs, TLS, inbox). Restore by replacing that directory and restarting.

**Upgrade:** `docker compose build && docker compose up -d`, or `./scripts/update-container.sh`.

## HTTPS and phones

Self-signed cert under `DATA_DIR/tls`, valid one year. Existing files are reused until SANs change. Phone camera needs `https://<lan-ip>:9443`. Accept the browser warning once.

`TLS_EXTRA_IPS` / `TLS_EXTRA_HOSTS` add SANs. `update-container.sh` can set `TLS_EXTRA_IPS` from a local route lookup if unset.

## Host network

If `curl http://127.0.0.1:9080/health` works on the server but other PCs time out on published ports, run the app on the host network:

```bash
docker compose -f docker-compose.yml -f deploy/host-network/docker-compose.host.yml up -d
```

Or copy that overlay to gitignored `docker-compose.override.yml`. The overlay points Ollama at `http://127.0.0.1:11434`. Published-port warnings from Compose are normal.

hp-scan already uses host network. Keep `HP_SCAN_HEALTH_PORT` off any other host port you care about (default **3001**).

## Reverse proxy

If you put a proxy in front, terminate TLS there and keep Sonix on a trusted LAN. The app does not implement SSO or proxy auth. Session cookie is `Secure` on HTTPS requests.

## Stuck extraction

On server start, jobs left in “processing” are marked failed (“Extraction interrupted (server restarted)”). Use **Retry extraction**. **Cancel extraction** aborts in-flight Ollama calls.
