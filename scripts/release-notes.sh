#!/bin/bash

set -euo pipefail

# Check if the required argument is provided
if [ $# -lt 1 ]; then
  echo "Usage: $0 <semver version> [<changelog file>]"
  exit 1
fi

version="$1"
changelog_file="${2:-CHANGELOG.md}"

# Check if the changelog file exists
if [ ! -f "$changelog_file" ]; then
  echo "Error: $changelog_file does not exist"
  exit 1
fi

CAPTURE=0
items=""
# Read the changelog file line by line
while IFS= read -r LINE; do
  # Stop capturing when we reach the next version sections
  if [[ "${LINE}" == "##"* ]] && [[ "${CAPTURE}" -eq 1 ]]; then
    break
  fi
  # Stop capturing when we reach the Unreleased section
  if [[ "${LINE}" == "[Unreleased]"* ]]; then
    break
  fi
  # Start capturing when we reach the specified version section
  if [[ "${LINE}" == "## [${version}]"* ]] && [[ "${CAPTURE}" -eq 0 ]]; then
    CAPTURE=1
    continue
  fi
  # Capture the lines between the specified version and the next version
  if [[ "${CAPTURE}" -eq 1 ]]; then
    # Ignore empty lines
    if [[ -z "${LINE}" ]]; then
      continue
    fi
    items+="$(echo "${LINE}" | xargs -0)"
    # Add a newline between each item
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
