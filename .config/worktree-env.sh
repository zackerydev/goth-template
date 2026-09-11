#!/usr/bin/env bash

set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
worktree_root=$(git -C "$repo_root" rev-parse --show-toplevel)
worktree_root=$(CDPATH= cd -- "$worktree_root" && pwd -P)
primary_root=$(git -C "$repo_root" worktree list --porcelain | sed -n '1s/^worktree //p')
primary_root=$(CDPATH= cd -- "$primary_root" && pwd -P)

worktree_name=$(basename "$worktree_root")
safe_name=$(printf '%s' "$worktree_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]/_/g; s/^_*//; s/_*$//')
if [ -z "$safe_name" ]; then
  safe_name=worktree
fi

if command -v shasum >/dev/null 2>&1; then
  path_hash=$(printf '%s' "$worktree_root" | shasum -a 256 | awk '{print substr($1, 1, 8)}')
else
  path_hash=$(printf '%s' "$worktree_root" | sha256sum | awk '{print substr($1, 1, 8)}')
fi
instance_slug=$(printf '%s_%s' "$safe_name" "$path_hash" | cut -c1-54)
port_seed=$((16#$path_hash))

requested_app_port=${APP_PORT:-${PORT:-}}
if [ "$worktree_root" = "$primary_root" ]; then
  default_app_port=8888
else
  default_app_port=$((10000 + port_seed % 40000))
fi
app_port=${requested_app_port:-$default_app_port}

validate_port() {
  case "$1" in
    ''|*[!0-9]*)
      printf 'Port must be an integer between 1 and 65535: %s\n' "$1" >&2
      return 1
      ;;
  esac
  if [ "$1" -lt 1 ] || [ "$1" -gt 65535 ]; then
    printf 'Port must be an integer between 1 and 65535: %s\n' "$1" >&2
    return 1
  fi
}

validate_port "$app_port"

port_is_occupied() {
  if command -v nc >/dev/null 2>&1; then
    nc -z -w 1 127.0.0.1 "$1" >/dev/null 2>&1
    return $?
  fi
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$1" <<'PY'
import socket
import sys

with socket.socket() as sock:
    sock.settimeout(1)
    sys.exit(0 if sock.connect_ex(("127.0.0.1", int(sys.argv[1]))) == 0 else 1)
PY
    return $?
  fi
  printf 'Neither nc nor python3 is available to check port availability.\n' >&2
  return 2
}

if [ "${APP_RUN_SMOKE:-0}" != 1 ] && port_is_occupied "$app_port"; then
  printf 'Port %s is already occupied; set APP_PORT or PORT.\n' "$app_port" >&2
  exit 1
fi

export APP_INSTANCE_SLUG="$instance_slug"
export APP_PORT="$app_port"
export APP_URL="http://127.0.0.1:$app_port"

printf 'Application instance: %s\n' "$APP_INSTANCE_SLUG"
printf 'Application: %s\n' "$APP_URL"
