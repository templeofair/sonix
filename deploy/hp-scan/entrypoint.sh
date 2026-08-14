#!/bin/sh
# Read printer IP written by Sonix Settings (Auto-import scans), then start the image init.
# Validate strictly — do not trust a hand-edited or compromised printer_ip file.
set -eu
STATE_DIR="${HP_SCAN_STATE_DIR:-/scan/hp-scan}"
IP_FILE="$STATE_DIR/printer_ip"
IP_WATCH="${HP_SCAN_IP_WATCH:-1}"
IP_WATCH_TICK="${HP_SCAN_IP_TICK:-3}"
IP_WATCH_DEBOUNCE="${HP_SCAN_IP_DEBOUNCE:-1}"
IP_WATCH_MIN_INTERVAL="${HP_SCAN_IP_MIN_INTERVAL:-30}"
SELFTEST="${HP_SCAN_SELFTEST:-0}"

# Strict IPv4 shape check, shared by startup and the watcher.
# 0 = valid, 2 = illegal characters, 3 = not a.b.c.d.
hp_ip_valid() {
  case "$1" in
    *[!0-9.]* | *..* | .* | *. | "") return 2 ;;
  esac
  [ "$(printf '%s' "$1" | tr -cd '.' | wc -c)" -eq 3 ] || return 3
}

# React to one kick per line on stdin: re-read the IP file, and restart the
# container (SIGTERM to s6 on PID 1) only when a valid, different IP is present.
hp_ip_react() {
  started_ip="$1"
  last_restart=0
  while read -r _event; do
    sleep "$IP_WATCH_DEBOUNCE"
    new_ip=''
    if [ -f "$IP_FILE" ]; then
      new_ip="$(tr -d '[:space:]' < "$IP_FILE" 2>/dev/null || true)"
    fi
    # Partially written or hand-edited garbage: wait for the next event.
    if [ -n "$new_ip" ] && ! hp_ip_valid "$new_ip"; then
      continue
    fi
    if [ "$new_ip" = "$started_ip" ]; then
      continue
    fi
    now="$(date +%s)"
    # A change inside the rate-limit window is applied by a later reconcile tick.
    if [ "$((now - last_restart))" -lt "$IP_WATCH_MIN_INTERVAL" ]; then
      continue
    fi
    last_restart="$now"
    echo "hp-scan: printer IP changed, restarting"
    if [ "$SELFTEST" = "1" ]; then
      echo "hp-scan: selftest: would signal PID 1 (printer IP=${new_ip:-<cleared>})"
      return 0
    fi
    # s6 stops its services cleanly, the container exits, and Docker's
    # restart policy brings it back so this entrypoint reads the new IP.
    kill -TERM 1 2>/dev/null || true
  done
}

# Kick source: inotify events on the directory (the file itself is removed when
# the operator clears the IP) plus a cheap reconcile tick, because inotify can be
# unavailable on some volume types and can drop events under queue overflow.
hp_ip_watch() {
  runtime="$(mktemp -d)"
  kick="$runtime/kick"
  mkfifo "$kick"
  ( while :; do sleep "$IP_WATCH_TICK"; echo tick; done ) > "$kick" 2>/dev/null &
  tick_pid=$!
  inotify_pid=''
  if [ "$1" = inotifyd ]; then
    inotifyd - "$STATE_DIR:ncdwmy" > "$kick" 2>/dev/null &
    inotify_pid=$!
  fi
  hp_ip_react "${IP:-}" < "$kick" || true
  kill "$tick_pid" ${inotify_pid:+"$inotify_pid"} 2>/dev/null || true
  rm -rf "$runtime"
}

if [ -f "$IP_FILE" ]; then
  FILE_IP="$(tr -d '[:space:]' < "$IP_FILE" || true)"
  if [ -n "$FILE_IP" ]; then
    # IPv4 only (OfficeJet LAN). Reject shell metacharacters and hostnames with odd chars.
    ip_rc=0
    hp_ip_valid "$FILE_IP" || ip_rc=$?
    if [ "$ip_rc" -eq 2 ]; then
      echo "hp-scan: invalid printer IP in $IP_FILE (IPv4 required)"
      sleep 30
      exit 1
    fi
    # Rough shape: a.b.c.d
    if [ "$ip_rc" -eq 3 ]; then
      echo "hp-scan: invalid printer IP in $IP_FILE (expected a.b.c.d)"
      sleep 30
      exit 1
    fi
    IP="$FILE_IP"
    export IP
  fi
fi
if [ -z "${IP:-}" ]; then
  echo "hp-scan: no printer IP yet. Set Printer IP in Sonix → Settings → Auto-import scans, then: docker compose --profile hp-scan up -d --force-recreate"
  sleep 30
  exit 1
fi
echo "hp-scan: using printer IP=${IP}"
if [ "$IP_WATCH" = "0" ]; then
  echo "hp-scan: IP watcher disabled (HP_SCAN_IP_WATCH=0)"
else
  if command -v inotifyd >/dev/null 2>&1; then
    WATCH_MECH=inotifyd
  else
    WATCH_MECH=reconcile-only
  fi
  echo "hp-scan: IP watcher active on $STATE_DIR (${WATCH_MECH}, ${IP_WATCH_TICK}s reconcile tick)"
  if [ "$SELFTEST" = "1" ]; then
    hp_ip_watch "$WATCH_MECH" || true
  else
    # Never fatal: if the watcher dies the helper keeps scanning on this IP.
    { hp_ip_watch "$WATCH_MECH" || true; echo "hp-scan: IP watcher stopped, helper stays on printer IP=${IP} (recreate the container to apply a new IP)"; } &
  fi
fi
if [ "$SELFTEST" = "1" ]; then
  exit 0
fi
# Inbox may be root-owned (Sonix app); helper runs as PUID/PGID and must write PDFs here.
mkdir -p /scan/inbox /scan/hp-scan
chown "${PUID:-1000}:${PGID:-1000}" /scan/inbox /scan/hp-scan 2>/dev/null || true
chmod u+rwx /scan/inbox /scan/hp-scan 2>/dev/null || true
mkdir -p /tmp/hp-scan
chown "${PUID:-1000}:${PGID:-1000}" /tmp/hp-scan 2>/dev/null || true
exec /init "$@"
