# terraform-provider-spoke

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)

Terraform provider for services offered by Spoke Platform. The repository is
currently a development foundation; the Go provider and service resources are
not implemented yet.

The [implementation plan](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
starts with stateful mock service endpoints and a complete bucket lifecycle,
then expands resource coverage and qualifies the provider against real platform
APIs. [HANDOFF.md](HANDOFF.md) records the original context: this repository owns
the provider, its dependency pins, and independent releases.

## Development

Install Nix with flakes enabled, then enter the pinned development environment:

```sh
nix develop
# The named shell is equivalent:
nix develop .#terraform
just
```

The shell provides Go, a C compiler for race tests, gopls, Go tools (including
goimports), Terraform, TFLint, tfplugindocs, actionlint, Just, nixfmt, Git, and shell utilities.
Terraform's license is allowed explicitly in the flake. Go toolchain downloads
are disabled, and acceptance tests use the Terraform executable from the shell.
Shell definitions cover Linux and macOS on x86_64 and aarch64; provider support
still requires qualification on each platform.

| Task | Purpose |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform, and Go source. |
| `just check` | Check formatting, workflows, and flake evaluation, then run Go vet and unit tests when `go.mod` exists. |
| `just build` | Build `bin/terraform-provider-spoke` from the future root Go module. |
| `just test` | Run unit tests with the race detector and acceptance tests disabled. |
| `just test-acc` | Run provider acceptance tests against an explicitly configured, isolated service fixture. |

Bare `just` lists these five tasks. Until the Go scaffold lands, `just check`
reports that provider checks are skipped; build and test tasks fail with an
explanation. Acceptance tests require
real service fixtures and externally supplied credentials, and may create or
delete resources. They are enabled only by `just test-acc`.

Keep `flake.lock` in version control. Update the nixpkgs revision in `flake.nix`
and run `nix flake lock` when intentionally upgrading the toolchain. Framework
and testing libraries will be pinned separately in `go.mod` and `go.sum` when
the provider is scaffolded. Provider packages and installation mirrors will be
added once there is a provider to build.

See [AGENTS.md](AGENTS.md) for development conventions and provider lifecycle
requirements. [CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)
evaluates every declared Nix system and runs `just check` on Linux for pull
requests and pushes to `main`. It also runs `just build` once `go.mod` exists.
Real service acceptance tests run separately through `just test-acc`.
