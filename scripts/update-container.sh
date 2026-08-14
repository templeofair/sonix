#!/usr/bin/env bash
# Rebuild the Docker image and restart the container (use after code changes).
# Always run this (or equivalent) before asking for manual UI/device checks.
#
# Compose: prefers `docker compose` (v2 plugin); falls back to `docker-compose` (v1).
#
# HP scan sidecar (Compose profile hp-scan):
#   ./scripts/update-container.sh --hp
#   WITH_HP_SCAN=1 ./scripts/update-container.sh
# Auto-on when HP_SCAN_LABEL or HP_SCAN_IP is set in the environment or .env,
# or when an hp-scan container already exists for this project.
# Host-network: copy deploy/host-network/docker-compose.host.yml to
# docker-compose.override.yml (gitignored) so this script merges it automatically.
# See docs/deployment.md.
#
# TLS_EXTRA_IPS: extra IPv4 SANs for the self-signed cert (phones). If unset, the
# script runs `ip -4 route get 1.1.1.1` — a local routing lookup, not a request
# to the internet — to find this host's LAN source address. Set TLS_EXTRA_IPS
# yourself to skip that lookup (air-gapped hosts or unusual routing).
set -euo pipefail
cd "$(dirname "$0")/.."

want_hp_scan=0
for arg in "$@"; do
  case "$arg" in
    --hp) want_hp_scan=1 ;;
    -h|--help)
      sed -n '2,19p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown option: $arg (use --hp for the HP scan profile)" >&2
      exit 1
      ;;
  esac
done

if docker compose version >/dev/null 2>&1; then
  compose=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose=(docker-compose)
else
  echo "error: need Docker Compose v2 (docker compose) or docker-compose (v1)" >&2
  exit 1
fi
echo "Compose: ${compose[*]}"

if [[ "${WITH_HP_SCAN:-}" == "1" ]]; then
  want_hp_scan=1
fi

# Explicit HP_* in env or project .env (compose default LABEL=Sonix does not count).
hp_env_set() {
  [[ -n "${HP_SCAN_LABEL:-}" || -n "${HP_SCAN_IP:-}" ]] && return 0
  if [[ -f .env ]]; then
    grep -Eq '^[[:space:]]*(export[[:space:]]+)?HP_SCAN_(LABEL|IP)=' .env
    return $?
  fi
  return 1
}

if [[ "$want_hp_scan" -eq 0 ]] && hp_env_set; then
  want_hp_scan=1
fi

# Detect before any rm — the wipe below used to remove sonix* including hp-scan.
if [[ "$want_hp_scan" -eq 0 ]] && docker ps -a --filter name=hp-scan --format '{{.Names}}' 2>/dev/null | grep -q .; then
  want_hp_scan=1
fi

compose_up=("${compose[@]}")
if [[ "$want_hp_scan" -eq 1 ]]; then
  compose_up+=(--profile hp-scan)
  echo "HP scan profile: on (--hp / WITH_HP_SCAN / HP_SCAN_* / existing helper)"
else
  echo "HP scan profile: off (pass --hp or set HP_SCAN_LABEL in .env to enable)"
fi

# Phones hit the host LAN IP; the cert inside Docker only sees the bridge IP unless we pass this.
if [[ -z "${TLS_EXTRA_IPS:-}" ]]; then
  detected="$(
    ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}'
  )"
  if [[ -n "${detected}" ]]; then
    export TLS_EXTRA_IPS="${detected}"
    echo "TLS_EXTRA_IPS=${TLS_EXTRA_IPS} (auto-detected for phone HTTPS)"
  fi
else
  echo "TLS_EXTRA_IPS=${TLS_EXTRA_IPS} (from environment; route lookup skipped)"
fi

echo "== ${compose[*]} build =="
"${compose[@]}" build

# docker-compose 1.29 + newer Docker Engine can fail recreate with KeyError: ContainerConfig.
# Removing the old app container first avoids that; the named volume is kept.
# Do not wipe hp-scan here — profiled sidecar is restored via --profile when wanted.
echo "== remove old app container (volume preserved) =="
"${compose[@]}" stop app 2>/dev/null || true
"${compose[@]}" rm -f app 2>/dev/null || true
docker rm -f sonix_app_1 2>/dev/null || true
# leftover app names from failed recreates (exclude hp-scan)
while read -r name id; do
  [[ -z "${id:-}" ]] && continue
  case "$name" in
    *hp-scan*) continue ;;
  esac
  docker rm -f "$id"
done < <(docker ps -a --filter name=sonix --format '{{.Names}} {{.ID}}' 2>/dev/null || true)

echo "== ${compose_up[*]} up -d =="
"${compose_up[@]}" up -d

echo "Container updated. Check: docker ps --filter name=sonix && curl -sI http://127.0.0.1:9080/health"
if [[ "$want_hp_scan" -eq 1 ]]; then
  echo "HP scan: docker ps --filter name=hp-scan"
fi
if [[ -n "${TLS_EXTRA_IPS:-}" ]]; then
  echo "Phone HTTPS: https://${TLS_EXTRA_IPS%%,*}:9443  (accept the self-signed warning once)"
  echo "Phone HTTP:  http://${TLS_EXTRA_IPS%%,*}:9080"
fi
