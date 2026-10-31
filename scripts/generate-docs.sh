#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

# tfplugindocs cleans its output directory. Keep it away from hand-maintained
# docs, including the roadmap. Its output flag is relative to the provider root.
generation_dir="$(mktemp -d "$repo_dir/.docs-generation.XXXXXX")"
trap 'rm -rf -- "$generation_dir"' EXIT
output_dir="$generation_dir/docs"
tfplugindocs generate --provider-name spoke \
    --rendered-website-dir "${generation_dir##*/}/docs"

# Validate the full output before changing any checked-in reference. Extend
# these explicit paths when the provider adds another generated category.
test -f "$output_dir/index.md"
test -d "$output_dir/resources"
test -d "$output_dir/data-sources"
mkdir -p docs
for entry in index.md resources data-sources; do
    rm -rf -- "docs/$entry"
    mv -- "$output_dir/$entry" "docs/$entry"
done
