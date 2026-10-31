#!/usr/bin/env bash
set -euo pipefail
for tool in go zig cc gcc clang; do
  if command -v "$tool" >/dev/null; then echo "Unexpected compiler: $tool" >&2; exit 1; fi
done
runtime="$(mktemp -d)"
mock-platform -directory "$runtime/fixture" > "$runtime/fixture.log" 2>&1 &
fixture_pid=$!
trap 'kill "$fixture_pid" 2>/dev/null || true; wait "$fixture_pid" 2>/dev/null || true' EXIT
for ((attempt=0; attempt<100; attempt++)); do
  [[ -f "$runtime/fixture/fixture.json" ]] && break
  kill -0 "$fixture_pid" 2>/dev/null || { cat "$runtime/fixture.log" >&2; exit 1; }
  sleep 0.1
done
[[ -f "$runtime/fixture/fixture.json" ]] || { echo 'Fixture did not start' >&2; exit 1; }
variables="$runtime/fixture/fixture.json"
terraform init -backend=false -input=false -lockfile=readonly
terraform apply -var-file="$variables" -input=false -auto-approve
terraform plan -var-file="$variables" -input=false -detailed-exitcode
id="$(terraform output -raw bucket_id)"
terraform state rm spoke_bucket.assets
terraform import -var-file="$variables" -input=false spoke_bucket.assets "$id"
terraform plan -var-file="$variables" -input=false -detailed-exitcode
terraform destroy -var-file="$variables" -input=false -auto-approve
printf 'Installed consumer passed; synthetic evidence: %s\n' "$runtime"
