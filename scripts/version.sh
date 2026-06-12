#!/usr/bin/env bash
# Manage the VERSION file (semver: MAJOR.MINOR.PATCH).
#
# Usage:
#   scripts/version.sh              # print current version
#   scripts/version.sh show
#   scripts/version.sh next         # version CI will tag on merge (reads VERSION)
#   scripts/version.sh bump patch   # bump and write VERSION
#   scripts/version.sh bump minor
#   scripts/version.sh bump major
#   scripts/version.sh set 1.2.3   # set explicit version

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="${ROOT}/VERSION"

die() {
  echo "version.sh: $*" >&2
  exit 1
}

read_version() {
  if [[ ! -f "$VERSION_FILE" ]]; then
    die "missing $VERSION_FILE"
  fi
  tr -d '[:space:]' < "$VERSION_FILE"
}

write_version() {
  local v="$1"
  validate_version "$v"
  printf '%s\n' "$v" > "$VERSION_FILE"
  echo "$v"
}

validate_version() {
  local v="$1"
  [[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid semver: $v (expected MAJOR.MINOR.PATCH)"
}

latest_tag() {
  git -C "$ROOT" tag -l 'v*' --sort=-v:refname 2>/dev/null | head -1
}

split_version() {
  local v="$1"
  validate_version "$v"
  IFS=. read -r MAJOR MINOR PATCH <<< "$v"
  echo "$MAJOR $MINOR $PATCH"
}

bump_part() {
  local v="$1" part="$2"
  read -r MAJOR MINOR PATCH <<< "$(split_version "$v")"
  case "$part" in
    patch) PATCH=$((PATCH + 1)) ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    *) die "unknown bump: $part (use patch, minor, or major)" ;;
  esac
  echo "${MAJOR}.${MINOR}.${PATCH}"
}

next_release_version() {
  read_version
}

cmd_show() {
  read_version
}

cmd_next() {
  next_release_version
}

cmd_bump() {
  local part="${1:-}"
  [[ -n "$part" ]] || die "usage: $0 bump <patch|minor|major>"
  write_version "$(bump_part "$(read_version)" "$part")"
}

cmd_set() {
  local v="${1:-}"
  [[ -n "$v" ]] || die "usage: $0 set <MAJOR.MINOR.PATCH>"
  write_version "$v"
}

usage() {
  sed -n '3,11p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

main() {
  local cmd="${1:-show}"
  shift || true
  case "$cmd" in
    -h|--help|help) usage 0 ;;
    show|"") cmd_show ;;
    next) cmd_next ;;
    bump) cmd_bump "${1:-}" ;;
    set) cmd_set "${1:-}" ;;
    *) die "unknown command: $cmd (run $0 --help)" ;;
  esac
}

main "$@"
