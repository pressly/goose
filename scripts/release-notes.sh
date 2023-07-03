#!/bin/bash

set -euo pipefail

# Check if the required argument is provided
if [ $# -lt 1 ]; then
  echo "Usage: $0 <semver version> [<changelog file>]"
  exit 1
fi

version="$1"
changelog_file="${2:-CHANGELOG.md}"

# Check if the CHANGELOG.md file exists
if [ ! -f "$changelog_file" ]; then
  echo "Error: $changelog_file does not exist"
  exit 1
fi

CAPTURE=0
items=""
while IFS= read -r LINE; do
    if [[ "${LINE}" == "##"* ]] && [[ "${CAPTURE}" -eq 1 ]]; then
        break
    fi
    if [[ "${LINE}" == "## [${version}]"* ]] && [[ "${CAPTURE}" -eq 0 ]]; then
        CAPTURE=1
        continue
    fi
    if [[ "${CAPTURE}" -eq 1 ]]; then
        items+="$(echo "${LINE}" | xargs)"
         # if items is not empty, add a newline
        if [[ -n "$items" ]]; then
          items+=$'\n'
        fi
    fi
done <"${changelog_file}"

if [[ -n "$items" ]]; then
    echo "${items%$'\n'}"
else
  echo "No changelog items found for version $version"
fi
