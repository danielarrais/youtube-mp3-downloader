#!/usr/bin/env sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output_dir="$project_dir/backend/build/windows/dist"

rm -rf "$output_dir"
mkdir -p "$output_dir"

docker build \
    --file "$project_dir/backend/build/windows/Dockerfile" \
    --target artifact \
    --output "type=local,dest=$output_dir" \
    "$project_dir"

installer=$(find "$output_dir" -maxdepth 1 -type f -name '*-installer.exe' | head -n 1)
if [ -z "$installer" ]; then
    echo "Windows installer was not generated." >&2
    exit 1
fi

echo "Windows installer: $installer"
