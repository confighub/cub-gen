#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "error: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
mkdir -p "$tmpdir/bin"

cat > "$tmpdir/bin/cub" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

case "$*" in
  "auth get-token")
    printf '%s\n' "test-token"
    ;;
  "context get --json")
    cat <<'JSON'
{
  "coordinate": {
    "serverURL": "https://hub.confighub.com",
    "user": "preflight-test@example.com"
  },
  "settings": {
    "defaultSpace": "platform"
  }
}
JSON
    ;;
  "space list")
    printf '%s\n' "[]"
    ;;
  *)
    echo "unexpected cub invocation: $*" >&2
    exit 1
    ;;
esac
SCRIPT

cat > "$tmpdir/bin/curl" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

out="/dev/null"
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      out="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

printf '%s' '{"message":"Not Found"}' > "$out"
printf '%s' "404"
SCRIPT

chmod +x "$tmpdir/bin/cub" "$tmpdir/bin/curl"

stdout="$tmpdir/stdout.txt"
stderr="$tmpdir/stderr.txt"
outdir="$tmpdir/out"

if PATH="$tmpdir/bin:$PATH" \
  SKIP_BUILD=1 \
  CONNECTED_FALLBACK_MODE=off \
  ./examples/demo/run-confighub-lifecycle-connected.sh \
    ./examples/helm-paas \
    ./examples/helm-paas \
    helm-paas \
    "$outdir" >"$stdout" 2>"$stderr"; then
  fail "expected bridge ingest preflight to fail on default 404"
fi

if ! grep -q "bridge ingest endpoint is not available" "$stderr"; then
  fail "missing actionable preflight failure; stderr was: $(cat "$stderr")"
fi

if [ -d "$outdir/create" ] || [ -d "$outdir/update" ]; then
  fail "preflight should fail before create/update lifecycle directories are written"
fi

echo "ok: connected lifecycle fails fast before bridge ingest bundle work when endpoint is missing"
