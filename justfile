set shell := ["bash", "-euo", "pipefail", "-c"]

# List the five public project tasks.
[default]
_default:
    @just --list

# Format Nix, Just, Terraform, and repository Go files.
fmt:
    nixfmt flake.nix
    just --unstable --fmt
    terraform fmt -recursive
    git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 -r gofmt -w

# Check tooling and formatting, then vet and unit test the provider when scaffolded.
check:
    #!/usr/bin/env bash
    set -euo pipefail
    nixfmt --check flake.nix
    just --unstable --fmt --check
    actionlint
    terraform fmt -check -recursive
    unformatted="$(git ls-files -z --cached --others --exclude-standard -- '*.go' | xargs -0 -r gofmt -l)"
    if [[ -n "$unformatted" ]]; then
        printf 'Run just fmt; unformatted Go files:\n%s\n' "$unformatted" >&2
        exit 1
    fi
    nix flake check
    if [[ -f go.mod ]]; then
        go vet ./...
        just test
    else
        echo "Tooling checks passed; provider checks skipped (no go.mod yet)."
    fi

# Build the provider executable for local development.
build:
    @test -f go.mod || { echo "Provider scaffold missing: add go.mod and main.go first." >&2; exit 1; }
    mkdir -p bin
    go build -o bin/terraform-provider-spoke .

# Run unit tests with the race detector and acceptance tests disabled.
test:
    @test -f go.mod || { echo "Provider scaffold missing: add go.mod and tests first." >&2; exit 1; }
    env -u TF_ACC go test -race -count=1 ./...

# Run real acceptance tests; requires an isolated service fixture and credentials.
test-acc:
    @test -f go.mod || { echo "Provider scaffold missing: add go.mod and acceptance tests first." >&2; exit 1; }
    TF_ACC=1 TF_ACC_TERRAFORM_PATH="$(command -v terraform)" go test -race -count=1 -v -timeout 30m ./internal/provider/...
