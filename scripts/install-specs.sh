#!/bin/bash
# Copy the hand-written specs into a tingly-box checkout (as _test.go files
# they never ship). usage: install-specs.sh <tingly-box checkout>
set -e
here=$(cd "$(dirname "$0")/.." && pwd)
dst=${1:?tingly-box checkout}
cd "$here/specs"
find . -name 'go2lean_specs_test.go' | while read -r f; do
  mkdir -p "$dst/$(dirname "$f")"
  cp "$f" "$dst/$f"
  echo "installed $f"
done
