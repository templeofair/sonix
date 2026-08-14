# Changelog

This project starts at the public snapshot. Private development history is not imported.

## Unreleased

### Added

- Initial public source snapshot: self-hosted letter scanner (Go API + React SPA + Docker Compose).
- Recorded Go and npm licence inventory in `THIRD-PARTY-NOTICES.md`.

### Changed

- Patched `golang.org/x/crypto` to 0.33 and `golang.org/x/image` to 0.24 (latest that still build on Go 1.22), plus PostCSS 8.5.26 and Autoprefixer 10.5.4. CI Actions stay on Node 20 / Go 1.22 with checkout/setup v7. Go 1.25, React 19, and Docker major/minor image bumps are deferred.
