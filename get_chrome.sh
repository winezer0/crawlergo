#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

: "${CHROME_VERSION:?Set CHROME_VERSION to a pinned Chrome for Testing version}"
: "${CHROME_SHA256:?Set CHROME_SHA256 to the expected archive SHA-256}"

if [[ ! "$CHROME_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid Chrome version: $CHROME_VERSION" >&2
  exit 1
fi

if [[ ! "$CHROME_SHA256" =~ ^[[:xdigit:]]{64}$ ]]; then
  echo "CHROME_SHA256 must contain 64 hexadecimal characters" >&2
  exit 1
fi

archive_url="https://storage.googleapis.com/chrome-for-testing-public/${CHROME_VERSION}/linux64/chrome-linux64.zip"
temp_dir="$(mktemp -d)"
trap 'rm -rf -- "$temp_dir"' EXIT

archive_path="${temp_dir}/chrome-linux64.zip"
curl --fail --location --retry 3 --retry-delay 2 --output "$archive_path" "$archive_url"
printf '%s  %s\n' "$CHROME_SHA256" "$archive_path" | sha256sum --check --status
unzip -q "$archive_path" -d "$temp_dir"

if [[ ! -x "${temp_dir}/chrome-linux64/chrome" ]]; then
  echo "Chrome archive is missing chrome-linux64/chrome" >&2
  exit 1
fi

rm -rf -- ./latest
mv "${temp_dir}/chrome-linux64" ./latest
