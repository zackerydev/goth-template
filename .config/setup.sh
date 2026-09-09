#!/usr/bin/env bash
# Initialize a new repo from this template.
# Usage: mise run init [-- owner/repo]  (prompts when omitted)
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$repo_root"

slug="${1:-${REPO:-}}"
if [ -z "$slug" ]; then
	printf 'GitHub repository (owner/repo): '
	if [ -r /dev/tty ]; then
		read -r slug < /dev/tty
	else
		read -r slug
	fi
fi

if [[ ! "$slug" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
	printf 'Expected owner/repo, got: %s\n' "$slug" >&2
	exit 1
fi

owner="${slug%%/*}"
name="${slug##*/}"
app=$(printf '%s' "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_-]/-/g; s/^[-_]*//; s/[-_]*$//')
if [ -z "$app" ]; then
	printf 'Could not derive an app name from: %s\n' "$slug" >&2
	exit 1
fi

old_module="github.com/zackerydev/goth-template"
new_module="github.com/$slug"

if command sed --version >/dev/null 2>&1; then
	sed_inplace=(sed -i)
else
	sed_inplace=(sed -i '')
fi

rewrite() {
	# $1 = search, $2 = replace; skips .git, worktrees, local state, and this script
	# (bash reads this file as it runs, so it must not rewrite itself).
	local files
	files=$(grep -RIl --exclude-dir=.git --exclude-dir=.worktrees --exclude-dir=.pi --exclude-dir=.rumdl_cache --exclude-dir=tmp --exclude-dir=bin --exclude=setup.sh "$1" . || true)
	[ -z "$files" ] && return 0
	printf '%s\n' "$files" | xargs "${sed_inplace[@]}" "s|$1|$2|g"
}

rewrite "$old_module" "$new_module"

if [ -d cmd/app ]; then
	if [ "$app" != "app" ]; then
		git mv cmd/app "cmd/$app" 2>/dev/null || mv "cmd/app" "cmd/$app"
	fi
fi
rewrite "cmd/app" "cmd/$app"

# Drop the template getting-started section from the README.
sed '/<!-- SETUP BEGIN -->/,/<!-- SETUP END -->/d' README.md > README.md.tmp
mv README.md.tmp README.md

if command -v go >/dev/null 2>&1; then
	go mod tidy
fi

printf '\nDone: %s (cmd/%s)\n' "$new_module" "$app"
printf 'Next: mise run setup && mise run dev\n'
