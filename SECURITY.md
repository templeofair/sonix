# Security policy

## Reporting

Email **me@hamidrezalaleh.com** or open a GitHub issue if the report does not include secrets.

Please do not file a public issue for a live credential, a private letter, or a working exploit.

## Threat model

Sonix is a **single-tenant, trusted-LAN** app.

- **One shared library.** Documents are not scoped per user. Any valid session can read, change, or delete every letter. There is no registration and no password-change UI.
- **Designed for a home LAN**, not the public internet. Do not publish ports to the world. There is no reverse-proxy auth in the app.
- **Self-signed HTTPS** on port 9443 exists so a phone on the same LAN can use the camera. Browsers will warn; that is expected.
- **Page images and text go to the Ollama URL you configure.** If that URL is on another machine, the documents leave this host. There is no telemetry.
- **Session cookie:** one `HttpOnly`, `SameSite=Strict` cookie, HMAC-signed with `SESSION_SECRET`. Login is rate-limited (8 attempts per IP per minute).
- **Inbox trust:** anything that can write to `DATA_DIR/inbox` can inject letters.
- **Backups** of `DATA_DIR` include page images, SQLite (including model output), and the TLS key.

## Operator duties

Set a long random `SESSION_SECRET` (`openssl rand -hex 32`). Do not use a placeholder. Seed a strong first password, then remove `SEED_PASSWORD` from `.env`. Keep Ollama on a host you trust.
