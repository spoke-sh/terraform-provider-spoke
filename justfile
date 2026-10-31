set shell := ["bash", "-euo", "pipefail", "-c"]

# List the five public project tasks.
[default]
_default:
    @just --list

# Format Nix, Just, Terraform, and repository Go files.
fmt:
    git ls-files -z --cached --others --exclude-standard -- '*.nix' | xargs -0 -r nixfmt
    just --unstable --fmt
    terraform fmt -recursive
    git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 -r gofmt -w

# Check formatting and workflows, vet Go, and run unit and mock acceptance tests.
check:
    #!/usr/bin/env bash
    set -euo pipefail
    git ls-files -z --cached --others --exclude-standard -- '*.nix' | xargs -0 -r nixfmt --check
    just --unstable --fmt --check
    actionlint
    terraform fmt -check -recursive
    unformatted="$(git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 -r gofmt -l)"
    if [[ -n "$unformatted" ]]; then
        printf 'Run just fmt; unformatted Go files:\n%s\n' "$unformatted" >&2
        exit 1
    fi
    nix flake check
    go vet ./...
    just test
    just test-acc mock

# Build the provider executable for local development.
build:
    mkdir -p bin
    go build -o bin/terraform-provider-spoke .

# Run unit tests with the race detector and acceptance tests disabled.
test:
    env -u TF_ACC go test -race -count=1 ./...

# Run mock acceptance, the Nix installed artifact, or explicit live qualification.
[positional-arguments]
test-acc target="mock":
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ "$1" == installed ]]; then
        exec ./scripts/test-installed.sh
    fi
    if [[ "$1" != mock ]]; then
        echo "Only the mock target is implemented. Live platform APIs are not qualified." >&2
        exit 2
    fi
    tests="$(go test -list '^TestAccMock' ./internal/provider)"
    if [[ "$tests" != *TestAccMock* ]]; then
        echo "No mock acceptance tests found." >&2
        exit 1
    fi
    TF_ACC=1 SPOKE_TEST_TARGET=mock TF_ACC_TERRAFORM_PATH="$(command -v terraform)" go test -race -count=1 -v -timeout 15m ./internal/provider -run '^TestAccMock'
