# terraform-provider-spoke

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)

Experimental Terraform provider for services offered by Spoke Platform. It
supports projects, environments, service accounts, grants, databases, buckets, artifact registries and scoped
discovery against the **provisional mock-v3 management API**. No live platform
service is qualified and no Registry release is available.

The [roadmap](docs/roadmap/README.md) tracks the completed foundation and remaining
milestones: contract and recovery coverage, the initial service graph, Nix
packaging, live qualification, releases and later services. It expands the
[implementation tracker](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
into ordered phases with explicit dependencies and acceptance criteria.
The [milestones 5–8 implementation plan](docs/roadmap/milestones-5-8-implementation-plan.md)
uses completed scope/access as the baseline for the remaining service graph.
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
credentials and isolated synthetic state in a private temporary directory. The
fixture persists acceptance before replying and can reload its ledger after a
service restart. `just test-acc live` currently fails
explicitly; live qualification requires a published platform contract and a
separately configured service fixture.

Keep `flake.lock` in version control. Update the nixpkgs revision in `flake.nix`
and run `nix flake lock` when intentionally upgrading the toolchain. Go libraries
are pinned in `go.mod` and `go.sum`. Nix provider packages, release mirrors, and
release automation remain follow-up work.

## Implemented slice

| Surface | Behavior |
| --- | --- |
| [`spoke_project`](docs/resources/project.md) | Organization-scoped project lifecycle, canonical import and protected parent deletion. |
| [`spoke_environment`](docs/resources/environment.md) | Project-scoped lifecycle, including a project created by the same organization-scoped configuration. |
| [`spoke_service_account`](docs/resources/service_account.md) | Machine identity lifecycle; credentials are issued externally and never returned to Terraform. |
| [`spoke_grant`](docs/resources/grant.md) | An explicitly owned read/manage grant to an exact target; independent grants survive its removal. |
| [`spoke_bucket`](docs/resources/bucket.md) | Capacity/retention settings, exact/collection discovery, protected empty-bucket deletion and compatible state upgrades. |
| [`spoke_database`](docs/resources/database.md) | Shared/dedicated service metadata, bounded capacity growth, retention and protected lifecycle. No SQL/migration/row operations or credential values. |
| [`spoke_artifact_registry`](docs/resources/artifact_registry.md) | OCI registry metadata, quota/retention, protected lifecycle and exact/collection discovery. Builds and uploads remain external. |
| [`data.spoke_artifact`](docs/data-sources/artifact.md) | Exact registry/repository lookup by digest or tag; returns an immutable digest without uploading content. |
| [`data.spoke_project`](docs/data-sources/project.md) | Discover the project explicitly selected in provider configuration. |
| [`data.spoke_environment`](docs/data-sources/environment.md) | Discover the selected environment and its parent scope. |
| Collection data sources | Complete bounded [`projects`](docs/data-sources/projects.md), [`environments`](docs/data-sources/environments.md), [`buckets`](docs/data-sources/buckets.md), [`service_accounts`](docs/data-sources/service_accounts.md) and [`grants`](docs/data-sources/grants.md) discovery. Denied or incomplete pages fail the read. |

See the [provider schema](docs/index.md), [examples](examples/), and
[mock-v3 contract](testdata/contracts/README.md) and
[capability matrix](testdata/contracts/capabilities.md). The provider always uses its
HTTP client; it has no mock mode or local provisioning implementation. The
fixture provides test endpoints and never creates real buckets or stores files.

Supply machine credentials through `SPOKE_TOKEN` or an external `token_file`.
Configure an explicit HTTPS endpoint and organization. Omit project/environment
assertions to bootstrap scope and access resources; an environment resource can
reference its new project's `resource_id`. Bucket resources require known project
and environment assertions. See the [bootstrap](examples/bootstrap/main.tf) and
[subsequent service consumer](examples/scoped-service/main.tf) examples. Provider
aliases can select different existing scopes; unknown configuration targets are
rejected before accessing the API.
A `ca_file` adds trusted certificates without disabling TLS verification.
Credential bytes are not resource attributes. Use a unique non-secret
`management_key` per resource and keep it stable across retries. Receipts and
resources are bound to the authenticated principal. Use distinct external
principals for independent owners. Import is not ownership transfer, and the
draft does not support adopting unmanaged objects or transferring ownership.

Bucket names may change without replacing their identity. Import IDs have the
form `organization_id/project_id/environment_id/bucket_id` for buckets,
`organization_id/project_id/environment_id` for environments, and
`organization_id/resource_id` for projects, service accounts and grants. Names
are never adoption keys.

Deletion protection defaults to true except for grants: apply
`deletion_protection = false` before destroying protected resources. Parents,
service accounts and data retain their dependency restrictions. There is no
force-empty or cascade operation. Removing a grant does not remove independent
grants. If self-revocation makes a completed operation unreadable, Terraform
preserves uncertainty and known state until inspection authority is restored.

The tests cover empty second plans, import, rename, drift, external deletion,
denied/incomplete reads, quotas, protected/dependency-restricted deletion,
concurrent owners, lost replies and pending operations. Linux installed-provider
tests cover four crash points across create/update/delete, fixture-ledger reload,
saved-plan revision conflicts, independent Terraform states, scope bootstrap,
provider aliases, self-revocation and secret canaries in state/plans/logs.
The consumers use checksum-locked filesystem mirrors with neither Go in their
PATH nor development overrides. See the
[milestones 2–4 qualification record](qualification/milestones-2-4.md).

Qualification so far is Go 1.26.7 and Terraform 1.15.3 on Linux amd64. Evaluation
of other Nix systems does not establish runtime support. Fixture restart reloads
a synthetic ledger within the test harness; it does not establish production
server/process or storage durability. Full service graphs, other Terraform
versions/platforms and live data-plane behavior remain unqualified. Sharing a
principal's credentials and management key across independent states remains
indistinguishable from legitimate recovery. Expired receipts leave tombstones
and fail explicitly rather than issuing a fresh mutation under an uncertain key.

Regenerate checked-in schema documentation after changing schemas/examples:

```sh
nix develop --command ./scripts/generate-docs.sh
```

The script replaces only the generated provider reference and preserves the
hand-maintained roadmap in `docs/roadmap/`.

See [AGENTS.md](AGENTS.md) for development conventions and provider lifecycle
requirements. [CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)
evaluates every declared Nix system and runs `just check` and `just build` on
Linux for pull requests and pushes to `main`. Mock acceptance is required in CI;
future live qualification will remain an explicitly selected separate lane.
