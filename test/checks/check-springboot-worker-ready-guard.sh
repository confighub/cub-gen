#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "error: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"

cleanup() {
  local pid_file="$tmpdir/project/var/worker.pid"
  if [ -f "$pid_file" ]; then
    kill "$(cat "$pid_file")" 2>/dev/null || true
  fi
  rm -rf "$tmpdir"
}
trap cleanup EXIT

mkdir -p "$tmpdir/bin" "$tmpdir/project/bin" "$tmpdir/project/var"
cp examples/springboot-paas/bin/install-worker "$tmpdir/project/bin/install-worker"

cat > "$tmpdir/bin/cub" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  space)
    [ "${2:-}" = "create" ] || exit 1
    ;;
  target)
    [ "${2:-}" = "create" ] || exit 1
    ;;
  worker)
    case "${2:-}" in
      run)
        printf '%s\n' "Sending Heartbeat acknowledged for test worker"
        while true; do sleep 1; done
        ;;
      list)
        cat <<'JSON'
[
  {
    "BridgeWorker": {
      "Condition": "Disconnected",
      "LastSeenAt": "2026-05-01T00:00:00Z"
    }
  }
]
JSON
        ;;
      *)
        echo "unexpected cub worker invocation: $*" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "unexpected cub invocation: $*" >&2
    exit 1
    ;;
esac
SCRIPT

chmod +x "$tmpdir/bin/cub" "$tmpdir/project/bin/install-worker"

stdout="$tmpdir/stdout.txt"
stderr="$tmpdir/stderr.txt"

if CUB="$tmpdir/bin/cub" \
  WORKER_READY_ATTEMPTS=2 \
  WORKER_READY_SLEEP_SECONDS=0 \
  "$tmpdir/project/bin/install-worker" test-cluster test-space >"$stdout" 2>"$stderr"; then
  fail "expected install-worker to fail while ConfigHub reports Disconnected"
fi

if ! grep -q "worker did not become Ready" "$stderr"; then
  fail "missing not-ready failure; stderr was: $(cat "$stderr")"
fi

if ! grep -q "heartbeat acknowledgements" "$stderr"; then
  fail "missing heartbeat acknowledgement diagnosis; stderr was: $(cat "$stderr")"
fi

if ! grep -q "cub-gen issue #219" "$stderr"; then
  fail "missing issue #219 stale-condition hint; stderr was: $(cat "$stderr")"
fi

if ! grep -q "2026-05-01T00:00:00Z" "$stderr"; then
  fail "missing LastSeenAt evidence; stderr was: $(cat "$stderr")"
fi

echo "ok: springboot install-worker fails loudly when active heartbeats still report Disconnected"
