#!/usr/bin/env bash
set -euo pipefail

# Build deterministic live-reconcile manifests from Helm example source state.
# Usage:
#   helm-live-reconcile-manifests.sh <create_repo_dir> <update_repo_dir> <output_root> [deployment_name]

trim_value() {
  local value="$1"
  value="${value%%#*}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  printf '%s' "$(printf '%s' "$value" | sed -E 's/[[:space:]]+$//')"
}

read_yaml_scalar() {
  local file="$1"
  local keypath="$2"
  if [ ! -f "$file" ]; then
    return 1
  fi

  if [[ "$keypath" != *.* ]]; then
    awk -v key="$keypath" '
      $0 ~ "^[[:space:]]*" key ":[[:space:]]*" {
        line=$0
        sub("^[[:space:]]*" key ":[[:space:]]*", "", line)
        print line
        exit
      }
    ' "$file"
    return 0
  fi

  local section="${keypath%%.*}"
  local key="${keypath#*.}"
  awk -v section="$section" -v key="$key" '
    $0 ~ "^[[:space:]]*" section ":[[:space:]]*$" {
      in_section=1
      next
    }
    in_section && $0 ~ "^[^[:space:]]" {
      in_section=0
    }
    in_section && $0 ~ "^[[:space:]]+" key ":[[:space:]]*" {
      line=$0
      sub("^[[:space:]]*" key ":[[:space:]]*", "", line)
      print line
      exit
    }
  ' "$file"
}

resolve_helm_value() {
  local repo_dir="$1"
  local keypath="$2"
  local value=""
  local candidate

  for candidate in "$repo_dir/values.yaml" "$repo_dir/values-prod.yaml" "$repo_dir/values-canary.yaml"; do
    if [ -f "$candidate" ]; then
      local found
      found="$(read_yaml_scalar "$candidate" "$keypath" || true)"
      if [ -n "$found" ]; then
        value="$(trim_value "$found")"
      fi
    fi
  done

  printf '%s' "$value"
}

render_manifest_set() {
  local repo_dir="$1"
  local output_dir="$2"
  local revision="$3"
  local deployment_name="$4"

  mkdir -p "$output_dir"

  local image_repo image_tag replicas service_port
  image_repo="$(resolve_helm_value "$repo_dir" "image.repository")"
  image_tag="$(resolve_helm_value "$repo_dir" "image.tag")"
  replicas="$(resolve_helm_value "$repo_dir" "replicaCount")"
  service_port="$(resolve_helm_value "$repo_dir" "service.port")"

  if [ -z "$image_repo" ] || [ -z "$image_tag" ] || [ -z "$replicas" ]; then
    echo "error: failed to resolve required Helm values from $repo_dir (image.repository/image.tag/replicaCount)." >&2
    return 1
  fi
  if [ -z "$service_port" ]; then
    service_port="8080"
  fi

  cat > "$output_dir/deployment.yaml" <<EOF_DEPLOY
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${deployment_name}
  labels:
    app.kubernetes.io/name: ${deployment_name}
    app.kubernetes.io/part-of: commerce-platform
    demo.confighub.io/revision: ${revision}
spec:
  replicas: ${replicas}
  selector:
    matchLabels:
      app: ${deployment_name}
  template:
    metadata:
      labels:
        app: ${deployment_name}
        demo.confighub.io/revision: ${revision}
    spec:
      containers:
        - name: ${deployment_name}
          image: ${image_repo}:${image_tag}
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: ${service_port}
EOF_DEPLOY

  cat > "$output_dir/kustomization.yaml" <<'EOF_KUSTOM'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml
EOF_KUSTOM

  jq -n \
    --arg revision "$revision" \
    --arg image "${image_repo}:${image_tag}" \
    --argjson replicas "$replicas" \
    --argjson service_port "$service_port" \
    '{revision: $revision, image: $image, replicas: $replicas, service_port: $service_port}' > "$output_dir/manifest-input.json"
}

build_helm_live_reconcile_manifests() {
  local create_repo_dir="$1"
  local update_repo_dir="$2"
  local output_root="$3"
  local deployment_name="${4:-payments-api}"

  if [ ! -d "$create_repo_dir" ]; then
    echo "error: create repo dir not found: $create_repo_dir" >&2
    return 1
  fi
  if [ ! -d "$update_repo_dir" ]; then
    echo "error: update repo dir not found: $update_repo_dir" >&2
    return 1
  fi

  local v1_dir v2_dir
  v1_dir="$output_root/manifests-v1"
  v2_dir="$output_root/manifests-v2"

  render_manifest_set "$create_repo_dir" "$v1_dir" "v1" "$deployment_name"
  render_manifest_set "$update_repo_dir" "$v2_dir" "v2" "$deployment_name"

  jq -n \
    --slurpfile v1 "$v1_dir/manifest-input.json" \
    --slurpfile v2 "$v2_dir/manifest-input.json" \
    --arg deployment_name "$deployment_name" \
    --arg path_v1 "$v1_dir" \
    --arg path_v2 "$v2_dir" \
    '{
      deployment_name: $deployment_name,
      create: $v1[0],
      update: $v2[0],
      path_v1: $path_v1,
      path_v2: $path_v2,
      update_applied: (($v1[0].image != $v2[0].image) or ($v1[0].replicas != $v2[0].replicas))
    }' > "$output_root/summary.json"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
    echo "usage: $0 <create_repo_dir> <update_repo_dir> <output_root> [deployment_name]" >&2
    exit 1
  fi
  build_helm_live_reconcile_manifests "$1" "$2" "$3" "${4:-payments-api}"
fi
