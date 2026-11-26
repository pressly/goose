#!/usr/bin/env bash

# Upgrade or check Go module dependencies.
#
#   go-deps.sh upgrade   upgrade all deps to latest, then apply the pins
#   go-deps.sh check     fail if a pin does not match
#
# A pin keeps a module at the exact version that another module requires in
# its own go.mod. For example, modernc.org/sqlite asks importers to use the
# same modernc.org/libc version as its go.mod. We also pin the libc runtime
# modules that sqlite requires, so the set stays the one sqlite tests with.
# Goose is a library, so these versions in our go.mod become the minimum for
# everyone who imports goose. See https://github.com/pressly/goose/issues/1150.
#
# The pins apply only to the root module, because it is the published one.

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

# Each entry is "<pinned module> <module whose go.mod sets the version>".
PINS=(
  "modernc.org/libc modernc.org/sqlite"
  "modernc.org/mathutil modernc.org/sqlite"
  "modernc.org/memory modernc.org/sqlite"
)

# Modules that are not published. They get upgrades, but no pins.
OTHER_MODULE_DIRS=(
  "./internal/testing"
)

# Each module must resolve on its own, without the local go.work file.
export GOWORK=off

usage() {
  echo "usage: ${0} upgrade|check" >&2
  exit 2
}

# Print the selected version of a module in the current module.
selected_version() {
  local module="${1}"
  go list -m -f '{{.Version}}' "${module}"
}

# Print the version of the pinned module that the source module requires in
# its go.mod.
required_version() {
  local pinned="${1}"
  local source="${2}"
  local source_version
  source_version="$(selected_version "${source}")"
  go mod graph |
    awk -v from="${source}@${source_version}" -v to="${pinned}@" \
      '$1 == from && index($2, to) == 1 { sub(to, "", $2); print $2 }'
}

# Print "<pinned> <want> <source>@<version>" for each pin.
pin_targets() {
  local pin pinned source want
  for pin in "${PINS[@]}"; do
    read -r pinned source <<<"${pin}"
    want="$(required_version "${pinned}" "${source}")"
    if [ -z "${want}" ]; then
      echo "error: ${source} does not require ${pinned}, update PINS in ${0}" >&2
      exit 1
    fi
    echo "${pinned} ${want} ${source}@$(selected_version "${source}")"
  done
}

check_pins() {
  local failed=0
  local targets pinned want source got
  targets="$(pin_targets)"
  while read -r pinned want source; do
    got="$(selected_version "${pinned}")"
    if [ "${got}" = "${want}" ]; then
      printf "ok    %-22s %-9s from %s\n" "${pinned}" "${got}" "${source}"
    else
      printf "FAIL  %-22s %-9s want %s from %s\n" \
        "${pinned}" "${got}" "${want}" "${source}"
      failed=1
    fi
  done <<<"${targets}"
  if [ "${failed}" -ne 0 ]; then
    echo "run 'make upgrade-deps' to fix the pins" >&2
  fi
  return "${failed}"
}

apply_pins() {
  local targets pinned want source
  targets="$(pin_targets)"
  while read -r pinned want source; do
    echo "==> pin ${pinned}@${want}"
    go get "${pinned}@${want}"
  done <<<"${targets}"
  go mod tidy
}

upgrade_module() {
  local module_dir="${1}"
  echo "==> upgrade ${module_dir}"
  (cd "${module_dir}" && go get -u -t ./... && go mod tidy)
}

main() {
  [ "${#}" -eq 1 ] || usage
  local module_dir
  case "${1}" in
    upgrade)
      upgrade_module "."
      apply_pins
      check_pins
      for module_dir in "${OTHER_MODULE_DIRS[@]}"; do
        upgrade_module "${module_dir}"
      done
      ;;
    check)
      check_pins
      ;;
    *)
      usage
      ;;
  esac
}

main "${@}"
