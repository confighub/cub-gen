#!/usr/bin/env bash
# Compare inventory-api config across dev, stage, prod.
#
# Shows a side-by-side table of key fields with divergence markers.
# Fields mutated in ConfigHub (diverging from upstream default) are marked *.
#
# Usage:
#   ./confighub-compare.sh              # Table output
#   ./confighub-compare.sh --explain    # What this script does (read-only)
#   ./confighub-compare.sh --json       # Machine-readable output
#
# Requires: jq. Optional: cub (falls back to local fixtures).
# Read-only: never mutates ConfigHub.

set -euo pipefail

CUB="${CUB:-cub}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ENVS=(dev stage prod)

show_explain() {
  cat <<'EOF'
confighub-compare: springboot-paas

Compares inventory-api operational config across dev, stage, and prod.

What it shows:
- Key deployment and config fields side by side
- Divergence markers (*) for fields whose live value differs from
  the upstream default for that environment
- Field route classification (mutable-in-ch / lift-upstream / generator-owned)

Read-only: fetches unit data but never mutates ConfigHub.
Falls back to local YAML fixtures when ConfigHub is not available.
EOF
}

if [[ "${1:-}" == "--explain" ]]; then
  show_explain
  exit 0
fi

JSON_OUTPUT=false
if [[ "${1:-}" == "--json" ]]; then
  JSON_OUTPUT=true
fi

command -v jq >/dev/null 2>&1 || { echo "error: jq not found." >&2; exit 1; }

# Field model: label, display name, route, upstream defaults, extraction method
FIELD_MODEL=$(cat <<'ENDJSON'
[
  {
    "label": "SPRING_PROFILES_ACTIVE",
    "display": "SPRING_PROFILES_ACTIVE",
    "route": "-",
    "upstream": { "dev": "default", "stage": "stage", "prod": "prod" },
    "extract": "env"
  },
  {
    "label": "reservationMode",
    "display": "feature.inventory.reservationMode",
    "route": "mutable-in-ch",
    "upstream": { "dev": "optimistic", "stage": "strict", "prod": "strict" },
    "extract": "configmap-grep"
  },
  {
    "label": "CACHE_BACKEND",
    "display": "CACHE_BACKEND",
    "route": "lift-upstream",
    "upstream": { "dev": "none", "stage": "none", "prod": "none" },
    "extract": "env"
  },
  {
    "label": "replicas",
    "display": "replicas",
    "route": "-",
    "upstream": { "dev": "1", "stage": "2", "prod": "3" },
    "extract": "replicas"
  },
  {
    "label": "containerPort",
    "display": "containerPort",
    "route": "-",
    "upstream": { "dev": "8080", "stage": "8080", "prod": "8081" },
    "extract": "port"
  },
  {
    "label": "SPRING_DATASOURCE_URL",
    "display": "SPRING_DATASOURCE_URL",
    "route": "generator-owned",
    "upstream": {
      "dev": "jdbc:postgresql://postgres.platform.svc:5432/inventory",
      "stage": "jdbc:postgresql://postgres.platform.svc:5432/inventory",
      "prod": "jdbc:postgresql://postgres.platform.svc:5432/inventory"
    },
    "extract": "env"
  }
]
ENDJSON
)

num_fields=$(echo "$FIELD_MODEL" | jq 'length')

# Parse multi-doc YAML into JSON array using awk (no python dependency)
yaml_to_json_array() {
  local yaml_file="$1"
  # Split on --- and parse each doc with jq-compatible yq or fall back
  if command -v yq >/dev/null 2>&1; then
    yq eval-all -o=json '[.]' "$yaml_file" 2>/dev/null || echo "[]"
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "
import sys, json, yaml
with open('${yaml_file}') as f:
    docs = list(yaml.safe_load_all(f))
json.dump([d for d in docs if d], sys.stdout)
" 2>/dev/null || echo "[]"
  else
    echo "[]"
  fi
}

extract_value() {
  local data="$1" extract_type="$2" field_label="$3"
  case "$extract_type" in
    env)
      echo "$data" | jq -r --arg n "$field_label" '
        [.[] | select(.kind == "Deployment")
         | .spec.template.spec.containers[]?.env[]?
         | select(.name == $n) | .value] | first // "-"
      ' 2>/dev/null
      ;;
    configmap-grep)
      local env_val
      env_val=$(echo "$data" | jq -r '
        [.[] | select(.kind == "Deployment")
         | .spec.template.spec.containers[]?.env[]?
         | select(.name == "FEATURE_INVENTORY_RESERVATIONMODE") | .value] | first // empty
      ' 2>/dev/null)
      if [[ -n "$env_val" ]]; then echo "$env_val"; return; fi
      local val
      val=$(echo "$data" | jq -r '
        [.[] | select(.kind == "ConfigMap")
         | .data["application.yaml"] // empty] | first // empty
      ' 2>/dev/null | grep "$field_label" | sed 's/.*: *//' || echo "-")
      echo "${val:-"-"}"
      ;;
    replicas)
      echo "$data" | jq -r '[.[] | select(.kind == "Deployment") | .spec.replicas] | first // "-"' 2>/dev/null
      ;;
    port)
      echo "$data" | jq -r '[.[] | select(.kind == "Deployment") | .spec.template.spec.containers[0].ports[0].containerPort] | first // "-"' 2>/dev/null
      ;;
    *)
      echo "-"
      ;;
  esac
}

# Fetch data per environment
if [[ "${JSON_OUTPUT}" != "true" ]]; then
  echo "Fetching unit data..."
  echo ""
fi

RESULT="[]"

for env in "${ENVS[@]}"; do
  data=""

  # Try ConfigHub first
  if command -v "${CUB}" >/dev/null 2>&1; then
    raw=$(${CUB} unit get --space "inventory-api-${env}" --data-only --json inventory-api 2>/dev/null || true)
    if [[ -n "$raw" && "$raw" != "null" ]]; then
      if echo "$raw" | jq -e '.Unit.Data' >/dev/null 2>&1; then
        b64=$(echo "$raw" | jq -r '.Unit.Data // empty')
        if [[ -n "$b64" ]] && command -v python3 >/dev/null 2>&1; then
          data=$(echo "$b64" | base64 -d 2>/dev/null | python3 -c "
import sys, json, yaml
docs = list(yaml.safe_load_all(sys.stdin))
json.dump([d for d in docs if d], sys.stdout)
" 2>/dev/null || true)
        fi
      fi
      if [[ -n "$data" ]]; then
        [[ "${JSON_OUTPUT}" != "true" ]] && echo "  ${env}: fetched from ConfigHub"
      fi
    fi
  fi

  # Fall back to local fixtures
  if [[ -z "$data" ]]; then
    yaml="${SCRIPT_DIR}/confighub/inventory-api-${env}.yaml"
    if [[ -f "$yaml" ]]; then
      data=$(yaml_to_json_array "$yaml")
      [[ "${JSON_OUTPUT}" != "true" ]] && echo "  ${env}: using local fixture"
    else
      [[ "${JSON_OUTPUT}" != "true" ]] && echo "  ${env}: no data available"
      continue
    fi
  fi

  for ((i=0; i<num_fields; i++)); do
    label=$(echo "$FIELD_MODEL" | jq -r ".[$i].label")
    extract=$(echo "$FIELD_MODEL" | jq -r ".[$i].extract")
    upstream=$(echo "$FIELD_MODEL" | jq -r --arg e "$env" ".[$i].upstream[\$e]")

    val=$(extract_value "$data" "$extract" "$label")
    val="${val:-"-"}"

    diverged="false"
    if [[ -n "$upstream" && "$val" != "$upstream" && "$val" != "-" ]]; then
      diverged="true"
    fi

    RESULT=$(echo "$RESULT" | jq \
      --arg env "$env" --argjson idx "$i" \
      --arg val "$val" --argjson div "$diverged" \
      '. + [{ env: $env, fieldIndex: $idx, value: $val, diverged: $div }]')
  done
done

[[ "${JSON_OUTPUT}" != "true" ]] && echo ""

# JSON output
if [[ "${1:-}" == "--json" ]]; then
  output="{}"
  for ((i=0; i<num_fields; i++)); do
    display=$(echo "$FIELD_MODEL" | jq -r ".[$i].display")
    route=$(echo "$FIELD_MODEL" | jq -r ".[$i].route")
    field_obj=$(echo "$RESULT" | jq --argjson idx "$i" --arg d "$display" --arg r "$route" '
      { display: $d, route: $r, values: (
          reduce (.[] | select(.fieldIndex == $idx)) as $item
            ({}; .[$item.env] = { value: $item.value, diverged: $item.diverged })
        ) }')
    output=$(echo "$output" | jq --arg d "$display" --argjson obj "$field_obj" '.[$d] = $obj')
  done
  echo "$output" | jq .
  exit 0
fi

# Table output
printf "%-38s %-15s %-15s %-15s %s\n" "Field" "dev" "stage" "prod" "Route"
printf "%-38s %-15s %-15s %-15s %s\n" \
  "──────────────────────────────────────" \
  "───────────────" "───────────────" "───────────────" \
  "───────────────"

for ((i=0; i<num_fields; i++)); do
  display=$(echo "$FIELD_MODEL" | jq -r ".[$i].display")
  route=$(echo "$FIELD_MODEL" | jq -r ".[$i].route")
  vals=()
  for env in "${ENVS[@]}"; do
    entry=$(echo "$RESULT" | jq -r --arg e "$env" --argjson idx "$i" \
      '[.[] | select(.env == $e and .fieldIndex == $idx)] | first // { value: "-", diverged: false }')
    val=$(echo "$entry" | jq -r '.value')
    div=$(echo "$entry" | jq -r '.diverged')
    if [[ ${#val} -gt 13 ]]; then val="${val:0:12}…"; fi
    if [[ "$div" == "true" ]]; then val="${val}*"; fi
    vals+=("$val")
  done
  printf "%-38s %-15s %-15s %-15s %s\n" "$display" "${vals[0]}" "${vals[1]}" "${vals[2]}" "$route"
done

echo ""
echo "* = diverges from upstream default (mutated in ConfigHub)"
echo ""
echo "Routes: mutable-in-ch (safe to edit) | lift-upstream (route to source) | generator-owned (blocked)"
