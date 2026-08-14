#!/bin/sh
# Host-side self-test for the hp-scan printer-IP watcher. No Docker, no printer.
# Runs deploy/hp-scan/entrypoint.sh with HP_SCAN_SELFTEST=1 against a temp state dir:
# the watcher logs the restart it would trigger instead of signalling PID 1, and
# the script does not exec /init.
#
# Usage: ./scripts/test-hp-ip-watch.sh
set -u

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ENTRYPOINT="$SCRIPT_DIR/../deploy/hp-scan/entrypoint.sh"
WORK="$(mktemp -d)"
CASES=0
FAILURES=0
WATCH_PID=''

RESTART_LINE='hp-scan: printer IP changed, restarting'
START_IP='192.168.1.50'

cleanup() {
  [ -n "$WATCH_PID" ] && kill "$WATCH_PID" 2>/dev/null
  sleep 1
  rm -rf "$WORK"
}
trap cleanup EXIT HUP INT TERM

pass() { CASES=$((CASES + 1)); echo "PASS: $1"; }
fail() { CASES=$((CASES + 1)); FAILURES=$((FAILURES + 1)); echo "FAIL: $1"; }

# Fresh state dir holding a valid starting IP.
new_case() {
  d="$WORK/$1"
  mkdir -p "$d/state" "$d/tmp"
  printf '%s\n' "$START_IP" > "$d/state/printer_ip"
  echo "$d"
}

start_watch() {
  d="$1"
  shift
  env TMPDIR="$d/tmp" HP_SCAN_STATE_DIR="$d/state" HP_SCAN_SELFTEST=1 \
    HP_SCAN_IP_TICK=1 HP_SCAN_IP_DEBOUNCE=1 "$@" \
    sh "$ENTRYPOINT" > "$d/log" 2>&1 &
  WATCH_PID=$!
}

has_line() { grep -qF -- "$2" "$1" 2>/dev/null; }

wait_for_line() {
  i=0
  while [ "$i" -lt "$3" ]; do
    has_line "$1" "$2" && return 0
    sleep 1
    i=$((i + 1))
  done
  return 1
}

wait_for_exit() {
  i=0
  while [ "$i" -lt "$2" ]; do
    kill -0 "$1" 2>/dev/null || return 0
    sleep 1
    i=$((i + 1))
  done
  return 1
}

stop_watch() {
  kill "$WATCH_PID" 2>/dev/null
  wait "$WATCH_PID" 2>/dev/null
  WATCH_PID=''
}

echo "hp-scan IP watcher self-test"
echo "entrypoint: $ENTRYPOINT"
echo

# 1. A valid, different IP triggers exactly one restart.
D="$(new_case detect)"
start_watch "$D"
sleep 2
printf '%s\n' '192.168.1.77' > "$D/state/printer_ip"
if wait_for_line "$D/log" "$RESTART_LINE" 10 &&
  has_line "$D/log" 'would signal PID 1 (printer IP=192.168.1.77)'; then
  pass "detects a valid new IP"
else
  fail "detects a valid new IP"
  cat "$D/log"
fi
if wait_for_exit "$WATCH_PID" 5 && wait "$WATCH_PID" 2>/dev/null; then
  pass "exits cleanly after signalling"
else
  fail "exits cleanly after signalling"
fi
if has_line "$D/log" 'hp-scan: IP watcher active' && has_line "$D/log" "hp-scan: using printer IP=$START_IP"; then
  pass "logs the startup lines"
else
  fail "logs the startup lines"
  cat "$D/log"
fi
WATCH_PID=''

# 2. Rewriting the same value must do nothing.
D="$(new_case same)"
start_watch "$D"
sleep 2
printf '%s\n' "$START_IP" > "$D/state/printer_ip"
printf '%s\n' "$START_IP" > "$D/state/printer_ip"
printf '%s\n' "  $START_IP  " > "$D/state/printer_ip"
sleep 5
if ! has_line "$D/log" "$RESTART_LINE" && kill -0 "$WATCH_PID" 2>/dev/null; then
  pass "ignores a rewrite of the same value"
else
  fail "ignores a rewrite of the same value"
  cat "$D/log"
fi
stop_watch

# 3. Invalid or partially written content must be ignored.
D="$(new_case invalid)"
start_watch "$D"
sleep 2
printf '%s\n' '192.168.1' > "$D/state/printer_ip"
sleep 2
printf '%s\n' '192.168.1.' > "$D/state/printer_ip"
sleep 2
printf '%s\n' 'printer.local; rm -rf /' > "$D/state/printer_ip"
sleep 3
if ! has_line "$D/log" "$RESTART_LINE" && kill -0 "$WATCH_PID" 2>/dev/null; then
  pass "ignores invalid content"
else
  fail "ignores invalid content"
  cat "$D/log"
fi
stop_watch

# 4. Removing the file (operator cleared the IP) must not crash the watcher.
D="$(new_case cleared)"
start_watch "$D"
sleep 2
rm -f "$D/state/printer_ip"
if wait_for_line "$D/log" "$RESTART_LINE" 10 &&
  has_line "$D/log" 'would signal PID 1 (printer IP=<cleared>)' &&
  ! grep -qiE 'no such file|not found|unexpected|syntax error' "$D/log"; then
  pass "handles file removal without crashing"
else
  fail "handles file removal without crashing"
  cat "$D/log"
fi
if wait_for_exit "$WATCH_PID" 5 && wait "$WATCH_PID" 2>/dev/null; then
  pass "exits cleanly after a cleared IP"
else
  fail "exits cleanly after a cleared IP"
fi
WATCH_PID=''

# 5. Kill switch reproduces today's behaviour: no watcher at all.
D="$(new_case killswitch)"
start_watch "$D" HP_SCAN_IP_WATCH=0
if wait_for_exit "$WATCH_PID" 5 &&
  has_line "$D/log" 'hp-scan: IP watcher disabled (HP_SCAN_IP_WATCH=0)' &&
  has_line "$D/log" "hp-scan: using printer IP=$START_IP" &&
  ! has_line "$D/log" 'hp-scan: IP watcher active'; then
  pass "HP_SCAN_IP_WATCH=0 starts no watcher"
else
  fail "HP_SCAN_IP_WATCH=0 starts no watcher"
  cat "$D/log"
fi
WATCH_PID=''

echo
if [ "$FAILURES" -eq 0 ]; then
  echo "PASS: $CASES/$CASES checks"
  exit 0
fi
echo "FAIL: $FAILURES of $CASES checks failed"
exit 1
