# terraform-provider-spoke

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)

Experimental Terraform provider for services offered by Spoke Platform. The
first implementation supports `spoke_bucket` and project/environment discovery
against a **provisional, mocked management API**. No live platform service is
qualified and no Registry release is available.

The [roadmap](roadmap/README.md) tracks the completed foundation and remaining
milestones: contract and recovery coverage, the initial service graph, Nix
packaging, live qualification, releases and later services. It expands the
[implementation tracker](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
into ordered phases with explicit dependencies and acceptance criteria.
[HANDOFF.md](HANDOFF.md) records the original context: this repository owns the
provider, its dependency pins, and independent releases.

## Development

Install Nix with flakes enabled, then enter the pinned development environment:

```sh
nix develop
# The named shell is equivalent:
nix develop .#terraform
just
```

The shell provides Go, a C compiler for race tests, gopls, Go tools (including
goimports), Terraform, TFLint, tfplugindocs, actionlint, Just, nixfmt, Git, and shell
utilities.
Terraform's license is allowed explicitly in the flake. Go toolchain downloads
are disabled, and acceptance tests use the Terraform executable from the shell.
Shell definitions cover Linux and macOS on x86_64 and aarch64; provider support
still requires qualification on each platform.

| Task | Purpose |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform, and Go source. |
| `just check` | Check formatting, workflows, and flake evaluation; run Go vet, unit tests, and mock Terraform acceptance tests. |
| `just build` | Build `bin/terraform-provider-spoke`. |
| `just test` | Run unit tests with the race detector and acceptance tests disabled. |
| `just test-acc` or `just test-acc mock` | Run real Terraform against automatically started local mock endpoints. |

Bare `just` lists these five tasks. Mock tests need no platform/cloud credentials,
Docker, or Transit runtime. Each test owns a local TLS server with synthetic
credentials and isolated in-memory state. `just test-acc live` currently fails
explicitly; live qualification requires a published platform contract and a
separately configured service fixture.

Keep `flake.lock` in version control. Update the nixpkgs revision in `flake.nix`
and run `nix flake lock` when intentionally upgrading the toolchain. Go libraries
are pinned in `go.mod` and `go.sum`. Nix provider packages, release mirrors, and
release automation remain follow-up work.

## Implemented slice

| Surface | Behavior |
| --- | --- |
| [`spoke_bucket`](docs/resources/bucket.md) | Create, refresh, update, import, and protected empty-bucket deletion; stable management keys and revision fences. |
| [`spoke_project`](docs/data-sources/project.md) | Discover the project explicitly selected in provider configuration. |
| [`spoke_environment`](docs/data-sources/environment.md) | Discover the selected environment and its parent scope. |

See the [provider schema](docs/index.md), [examples](examples/), and
[mock-v1 contract](testdata/contracts/README.md). The provider always uses its
HTTP client; it has no mock mode or local provisioning implementation. The
fixture provides test endpoints and never creates real buckets or stores files.

Supply machine credentials through `SPOKE_TOKEN` or an external `token_file`.
Configure an explicit HTTPS endpoint, organization, project, and environment.
A `ca_file` adds trusted certificates without disabling TLS verification.
Credential bytes are not resource attributes. Use a unique non-secret
`management_key` per resource and keep it stable across retries.

Bucket names may change without replacing their identity. Import IDs have the
form `organization_id/project_id/environment_id/bucket_id`. Deletion protection
defaults to true: apply `deletion_protection = false` before destroying an empty
bucket. There is no force-empty or cascade operation.

The tests cover empty second plans, import, rename, drift, external deletion,
denied reads, nonempty/protected deletion, stale revisions, lost replies, pending
operations, and concurrent identical creates. On Linux, a separate-process test
installs a built binary through a filesystem mirror with checksum locking,
kills it after server acceptance, and recovers the same bucket on the next
apply. That consumer has neither Go in its PATH nor development overrides.

Qualification so far is Go 1.26.7 and Terraform 1.15.3 on Linux amd64. Evaluation
of other Nix systems does not establish runtime support. The mock ledger is
in-memory; server-restart durability, full service graphs, other Terraform
versions/platforms, and live data-plane behavior remain unqualified. Sharing a
management key and credentials across independent states grants the same
authority; this draft protocol cannot distinguish that from crash recovery.

Regenerate checked-in schema documentation after changing schemas/examples:

```sh
nix develop --command tfplugindocs generate --provider-name spoke
```

See [AGENTS.md](AGENTS.md) for development conventions and provider lifecycle
requirements. [CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)
evaluates every declared Nix system and runs `just check` and `just build` on
Linux for pull requests and pushes to `main`. Mock acceptance is required in CI;
future live qualification will remain an explicitly selected separate lane.
