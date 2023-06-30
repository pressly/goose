#!/bin/sh
# Adapted from the Deno installer: Copyright 2019 the Deno authors. All rights reserved. MIT license.
# Ref: https://github.com/denoland/deno_install
# TODO(everyone): Keep this script simple and easily auditable.

# TODO(mf): this should work on Linux and macOS. Not intended for Windows.

set -e

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

if [ "$arch" = "aarch64" ]; then
	arch="arm64"
fi

# Set default version to v3.11.2
default_version="v3.11.2"
next_version="v3.13.0"

# Always display the warning message
echo "Warning: For versions v3.13.0 or later, please consider using 'go install' instead."

if [ $# -eq 0 ]; then
	version="${default_version}"
else
	version="${1}"
	# Check if the specified version is greater than or equal to v3.13.0
	if [ "$(printf '%s\n' "${version}" "${next_version}" | sort -V | tail -n1)" = "${version}" ]; then
		echo "Error: Specified version is v3.13.0 or later. Please specify a version earlier than v3.13.0."
		exit 1
	fi
fi

goose_uri="https://github.com/pressly/goose/releases/download/${version}/goose_${os}_${arch}"

goose_install="${GOOSE_INSTALL:-/usr/local}"
bin_dir="${goose_install}/bin"
exe="${bin_dir}/goose"

if [ ! -d "${bin_dir}" ]; then
	mkdir -p "${bin_dir}"
fi

curl --silent --show-error --location --fail --location --output "${exe}" "$goose_uri"
chmod +x "${exe}"

echo "Goose was installed successfully to ${exe}"
if command -v goose >/dev/null; then
	echo "Run 'goose --help' to get started"
fi
