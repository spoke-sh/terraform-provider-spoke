# terraform-provider-spoke

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)

Experimental Terraform provider for services offered by Spoke Platform. It
supports scope/access, databases, buckets, artifact registries, secret references,
service bindings, containers, endpoints, jobs/schedules, streams/queues, volumes,
domains/private connections, backups/clones, scaling/placement and scoped discovery against the
**provisional mock-v5 management API**. No live platform
service is qualified and no Registry release is available.

The [roadmap](docs/roadmap/README.md) tracks the completed foundation and remaining
milestones: contract and recovery coverage, the initial service graph, Nix
packaging, live qualification, releases and later services. It expands the
[implementation tracker](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
into ordered phases with explicit dependencies and acceptance criteria.
The [milestones 5–8 implementation plan](docs/roadmap/milestones-5-8-implementation-plan.md)
records the service graph design and its qualification gates. The
[milestones 9–11 plan](docs/roadmap/milestones-9-11-implementation-plan.md) covers
Nix packaging, native/live qualification and release distribution.
The [remaining milestones 10–17 plan](docs/roadmap/milestones-10-17-implementation-plan.md)
refreshes the upstream assessment against Transit `a000b8d` and Platform's native
management v2, with SDK/service prerequisites, logical commits and completion gates.
The [10–17 execution record](qualification/milestones-10-17.md) tracks the delivered
codecs, mock-v5 services and remaining live blockers.
[HANDOFF.md](HANDOFF.md) records the original context: this repository owns the
provider, its dependency pins, and independent releases.

The explicit [native v2 profile](docs/native-v2.md) integrates the platform's
seeded bucket metadata boundary. It has separate signed credentials and durable
dispatch coordination; it does not qualify object storage or the full service graph.

## Development

Install Nix with flakes enabled, then enter the pinned development environment:

```sh
nix develop
# The named shell is equivalent:
nix develop .#terraform
just
```

The shell provides Go, a C compiler for race tests, gopls, Go tools (including
goimports), Terraform, TFLint, tfplugindocs, actionlint, Just, nixfmt, Git, Python,
GnuPG, the GitHub CLI, and shell utilities.
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
| `just test-acc [mock/installed/live]` | Run mock tests, the exact Nix artifact, or the explicit native prerequisite gate. |

Bare `just` lists these five tasks. Mock tests need no platform/cloud credentials,
Docker, or Transit runtime. Each test owns a local TLS server with synthetic
credentials and isolated synthetic state in a private temporary directory. The
fixture persists acceptance before replying and can reload its ledger after a
service restart. `just test-acc installed` exercises the Nix artifact through
the same lifecycle and recovery assertions. `just test-acc live` requires explicit
configuration and emits a blocked report until the selected target cases execute;
it cannot silently pass or fall back to mocks.

Keep `flake.lock` in version control. Update the nixpkgs revision in `flake.nix`
and run `nix flake lock` when intentionally upgrading the toolchain. Go libraries
are pinned in `go.mod` and `go.sum`. Linux amd64 provider/mirror packages and a
separate synthetic fixture package are available. See the
[distribution guide](docs/distribution.md) and
[independent pinned consumer](examples/installed-consumer/README.md). Release
candidates reuse exact mirror bytes; public releases require the full live gate.

## Implemented slice

| Surface | Behavior |
| --- | --- |
| [`spoke_project`](docs/resources/project.md) | Organization-scoped project lifecycle, canonical import and protected parent deletion. |
| [`spoke_environment`](docs/resources/environment.md) | Project-scoped lifecycle, including a project created by the same organization-scoped configuration. |
| [`spoke_service_account`](docs/resources/service_account.md) | Machine identity lifecycle; credentials are issued externally and never returned to Terraform. |
| [`spoke_grant`](docs/resources/grant.md) | An explicitly owned management or service-specific runtime grant to an exact target; independent grants survive its removal. |
| [`spoke_bucket`](docs/resources/bucket.md) | Capacity/retention settings, exact/collection discovery, protected empty-bucket deletion and safe endpoint observations. |
| [`spoke_database`](docs/resources/database.md) | Shared/dedicated service metadata, bounded capacity growth, retention and protected lifecycle. No SQL/migration/row operations or credential values. |
| [`spoke_artifact_registry`](docs/resources/artifact_registry.md) | OCI registry metadata, quota/retention, protected lifecycle and exact/collection discovery. Builds and uploads remain external. |
| [`data.spoke_artifact`](docs/data-sources/artifact.md) | Exact registry/repository lookup by digest or tag; returns an immutable digest without uploading content. |
| [`spoke_secret_reference`](docs/resources/secret_reference.md) | External owner/identifier/pinned version metadata; no secret payloads or credential issuance. |
| [`spoke_service_binding`](docs/resources/service_binding.md) | Exact consumer/target/slot connection intent; explicit runtime grants remain separate. |
| [`spoke_container_service`](docs/resources/container_service.md) | Immutable digest, explicit runtime identity, bounded configuration and rollout observations. |
| [`spoke_endpoint`](docs/resources/endpoint.md) | Explicit private/public exposure, managed TLS and routing observations. |
| [`data.spoke_project`](docs/data-sources/project.md) | Discover the project explicitly selected in provider configuration. |
| [`data.spoke_environment`](docs/data-sources/environment.md) | Discover the selected environment and its parent scope. |
| [`spoke_job`](docs/resources/job.md) / [`spoke_schedule`](docs/resources/schedule.md) | Pinned job definitions, bounded admission and explicit cron/time zone; executions remain scheduler observations. |
| [`spoke_stream`](docs/resources/stream.md) / [`spoke_queue`](docs/resources/queue.md) | Messaging capacity, retention and delivery settings; explicit runtime bindings/grants. |
| [`spoke_volume`](docs/resources/volume.md) / [`spoke_volume_attachment`](docs/resources/volume_attachment.md) | Retained volume declarations, exact mounts and observed writer fences. |
| [`spoke_custom_domain`](docs/resources/custom_domain.md) / [`spoke_private_connection`](docs/resources/private_connection.md) | External domain proof, observed TLS and independent peer consent. |
| [`spoke_backup_policy`](docs/resources/backup_policy.md) / [`spoke_database_clone`](docs/resources/database_clone.md) | Scheduled backup intent and exact frozen recovery-point selection; no backup contents in state. |
| [`spoke_scaling_policy`](docs/resources/scaling_policy.md) / [`spoke_placement_policy`](docs/resources/placement_policy.md) | Terraform owns constraints; controllers own observed count/location. |
| Execution / recovery history | [`job_executions`](docs/data-sources/job_executions.md) and [`recovery_points`](docs/data-sources/recovery_points.md) provide complete bounded metadata without starting work. |
| Exact service data sources | All 19 environment-scoped service kinds expose typed settings and safe observations without adopting ownership. |
| Collection data sources | All 23 resource kinds support complete bounded discovery, including [`projects`](docs/data-sources/projects.md), [`environments`](docs/data-sources/environments.md), [`buckets`](docs/data-sources/buckets.md), [`service_accounts`](docs/data-sources/service_accounts.md) and [`grants`](docs/data-sources/grants.md) discovery. Denied or incomplete pages fail the read. |

See the [provider schema](docs/index.md), [examples](examples/), and
[mock-v5 contract](testdata/contracts/README.md) and
[capability matrix](testdata/contracts/capabilities.md). The provider always uses its
HTTP client; it has no mock mode or local provisioning implementation. The
fixture provides test endpoints and synthetic in-memory data; it never provisions
real customer services. The [later-family contract](testdata/contracts/lifecycles.md)
records explicit limits for each mock profile. There are 23 resources and 47 data
sources; all remain experimental.

Supply machine credentials through `SPOKE_TOKEN` or an external `token_file`.
Configure an explicit HTTPS endpoint and organization. Omit project/environment
assertions to bootstrap scope and access resources; an environment resource can
reference its new project's `resource_id`. Environment-scoped services require known project
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

Service names may change without replacing their identity. Import IDs have the
form `organization_id/project_id/environment_id/resource_id` for services,
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
[milestones 2–4 qualification record](qualification/milestones-2-4.md) and the
[milestones 5–8 record](qualification/milestones-5-8.md). Current mock-v5, Nix and
release-candidate results are in the [9–11 record](qualification/milestones-9-11.md);
the [native boundary assessment](qualification/native-boundary.md) records the
remaining live prerequisites.

Qualification so far is Go 1.26.7 and Terraform 1.15.3 on Linux amd64. Evaluation
of other Nix systems does not establish runtime support. Fixture restart reloads
a synthetic ledger within the test harness; it does not establish production
server/process or storage durability. Other Terraform versions/platforms and
live data-plane behavior remain unqualified. Sharing a
principal's credentials and management key across independent states remains
indistinguishable from legitimate recovery. Expired receipts leave tombstones
and fail explicitly rather than issuing a fresh mutation under an uncertain key.

## Application graph and contract cutovers

The [complete service consumer](examples/service-graph/README.md) uses concrete
bootstrap outputs, explicit runtime grants, exact connection slots, a pinned
image digest and a pinned external secret version. Builds/uploads and secret
issuance remain external. A binding never grants access. Non-secret configuration
is stored in Terraform state; secret references contain only identifiers.

Apply completes durable declarations. `readiness`, `reason`, `desired_revision`,
`effective_revision`, `active_revision` and endpoint `tls_state` separately report
controller observations. Failed rollout keeps the desired intent and last active
revision; successful apply does not prove healthy traffic. Reads do not advance
controllers. Setting the example's `deploy = false` retires endpoints, bindings
and compute while retaining protected services, artifacts and independent grants.
Database capacity can grow; shrink is rejected. Profile changes require protected
replacement, and nonempty services require an external migration/cleanup workflow.

Development contracts use hard cutovers, with no backwards compatibility until
all roadmap milestones 1–17 are complete. Keep provider, fixture and configuration
pinned together; new contract tests use fresh isolated state. Preserve uncertain
work and retained services rather than treating an incompatible version as absence.
Published artifact versions remain immutable.

mock-v5 removes previous state/ledger migrations and legacy wire defaults.
Resource schema version 3 rejects prior state before mutation.
The earlier [qualification record](qualification/milestones-5-8.md) is historical evidence,
not a promise to maintain migration support in subsequent development versions.

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
