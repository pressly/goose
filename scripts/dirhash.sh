#!/bin/bash

set -euo pipefail

# Check if the required argument is provided, a directory path
if [ $# -lt 1 ]; then
  echo "Usage: $0 <directory path>"
  exit 1
fi

dir_path="$1"

# Check if the directory path exists
if [ ! -d "$dir_path" ]; then
  echo "Error: $dir_path does not exist"
  exit 1
fi

# Calculate the hash of each directory in the specified directory
for dir in "$dir_path"/*; do
  if [ -d "$dir" ]; then
    # remove the dir_path from the found files
    all_files=$(find "$dir" -type f | sort | sed "s|$dir_path/||")
    cd $dir_path
    sha256sum $all_files
    digest=$(sha256sum $all_files | sha256sum | cut -c 1-32)
    echo ""
    echo "Hash of $dir: $digest"
    echo $digest > "$(basename "$dir").sha256"
    echo ""
  fi
done
