# Troubleshooting

## App is up on the server, other machines time out

`curl http://127.0.0.1:9080/health` on the host returns 200, but phones/PCs get connection timed out. Bridge-published ports are not reaching the LAN. Use the [host-network overlay](deployment.md#host-network). Remapping the host port alone does not fix that class of failure.

## Certificate warning on the phone

Expected for the self-signed cert. Use `https://<lan-ip>:9443`, accept once. If the IP is new, set `TLS_EXTRA_IPS` or rerun `./scripts/update-container.sh`, then accept again. To mint a new cert, delete `DATA_DIR/tls` and restart.

## Cannot reach Sonix in the UI

The SPA shows that screen when the session probe fails (wrong host, container down, mixed HTTP/HTTPS). Confirm `docker ps` lists the app as Up and `/health` is 200.

## Extraction stuck in progress

Restart marks interrupted jobs failed. **Cancel extraction**, then **Extract now** / **Retry**. Partial means original text was saved; Retry finishes translation/summary.

## Ollama unreachable / model missing

The app still serves the library. Extraction fails with a short UI error; details are in server logs. `ollama pull` the tags in `.env`. Test from Settings. `OLLAMA_BASE_URL` must be http(s); link-local and credential URLs are rejected.

## Slow extraction

Default is one document at a time. Keep models loaded (`OLLAMA_KEEP_ALIVE` on the Ollama host). See [performance.md](performance.md). Raise `OLLAMA_TIMEOUT_MINUTES` / `EXTRACTION_JOB_TIMEOUT_MINUTES` on slow CPUs — do not drop them so low that good jobs fail.

## Login rejected / seed user missing

`SESSION_SECRET` must be set and not a placeholder. Seed password must be at least 12 characters, and only runs when the user table is empty. Login: 8 attempts per IP per minute.

## Compose v1 vs v2

This host may only have `docker-compose`. `update-container.sh` tries `docker compose` then falls back. Prefer Compose v2 when you can.
