#!/usr/bin/env bash

# Compiles the Swift EventKit helper used by the apple-calendar panel.

set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v swiftc >/dev/null 2>&1; then
  echo "swiftc not found - skipping calendar helper build (calendar panel will not work)" >&2
  exit 0
fi

mkdir -p bin

# Skip if the helper is already up to date.
if [ bin/apple-calendar-helper -nt swift/apple-calendar-helper.swift ]; then
  exit 0
fi

swiftc -O -o bin/apple-calendar-helper swift/apple-calendar-helper.swift
