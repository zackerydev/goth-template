#!/usr/bin/env bash

set -euo pipefail

root_commit=$(git rev-list --max-parents=0 HEAD)

while IFS= read -r commit; do
  message=$(git show --no-patch --format=%B "$commit")

  # GitHub creates repositories from templates with this fixed root message.
  if [ "$commit" = "$root_commit" ] && [ "$message" = "Initial commit" ]; then
    continue
  fi

  printf '%s\n' "$message" | cog --config .config/cog.toml verify --ignore-merge-commits --file -
done < <(git rev-list --reverse HEAD)
