#!/usr/bin/env bash
set -euo pipefail

# Build once, then inject the immutable mirror into every installed-process test.
# No test in this lane may silently compile a replacement provider.
mirror="$(nix build --no-update-lock-file --no-link --print-out-paths .#terraform-provider-spoke-mirror)"
binaries=("$mirror"/registry.terraform.io/spoke-sh/spoke/*/linux_amd64/terraform-provider-spoke_v*)
[[ ${#binaries[@]} == 1 && -x "${binaries[0]}" ]] || { echo 'Expected exactly one Linux amd64 artifact' >&2; exit 1; }
version="$(basename "$(dirname "$(dirname "${binaries[0]}")")")"
export SPOKE_TEST_PROVIDER_MIRROR="$mirror" SPOKE_TEST_PROVIDER_VERSION="$version"
export TF_ACC=1 SPOKE_TEST_TARGET=mock
export TF_ACC_TERRAFORM_PATH="$(command -v terraform)"
"${binaries[0]}" -version
sha256sum "${binaries[0]}"
cases="$(go test -list '^TestAccMock' ./internal/provider)"
[[ "$cases" == *TestAccMockInstalledServiceGraph* && "$cases" == *TestAccMockProviderProcessRecovery* && "$cases" == *TestAccMockInstallationFailures* ]] || { echo 'Missing required artifact cases' >&2; exit 1; }
go test -race -count=1 -v -timeout 35m ./internal/provider -run '^TestAccMock'
