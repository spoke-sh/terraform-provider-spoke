# Spoke Terraform provider implementation handoff

Prepared 2026-10-31 for the next agent working in this repository.

Build a Terraform provider that lets other repositories declare **services
offered by Spoke Platform**: artifact registries, databases, buckets and container
services, with explicit access and inspectable lifecycle. This repository is the
provider's implementation, test, documentation, Nix and release owner.

The immediate work is to establish the provider foundation and verify the
platform management API needed for the first real resource. Customer service
provisioning is still planned in the inspected platform source. A working
browser integration does not establish those management APIs.

## Decisions from the conversation

- Use `spoke-sh/terraform-provider-spoke` as a separate repository. This replaces
  the original proposal to implement the provider inside `spoke-sh/transit`.
- The intended Terraform provider type is `spoke`, source address
  `spoke-sh/spoke`, with resource names such as `spoke_database`. Registry
  registration and availability have not been verified.
- The user specifically requested artifact registries, databases and containers,
  and asked about buckets/files. The recommended initial offering includes
  buckets plus the ownership, identity and networking resources they need.
  Exact schemas and the later service inventory remain design work.
- Files normally belong in buckets as objects or in registries as immutable
  artifacts. Persistent filesystem/block volumes are a separate service.
- Nix should support both installing/developing the provider and building
  application artifacts that Terraform subsequently deploys by immutable digest.
- Give the provider independent SemVer releases and a tested platform API
  compatibility matrix. Do not synchronize its version with Transit's `VERSION`.
- Implementation planning belongs in GitHub tracking issues. This file is an
  agent handoff requested by the user, not a replacement roadmap or API contract.

## Repository ownership

| Repository | Responsibility |
| --- | --- |
| `spoke-sh/terraform-provider-spoke` | Terraform schemas, resource lifecycle adapter, platform API client integration, tests, examples, generated docs, Nix packaging and provider releases |
| `spoke-sh/platform` | Customer service definitions, management APIs, tenancy, product policy, provisioning/reconciliation and service guarantees |
| `spoke-sh/transit` | Generic database, authorization, storage, execution, serving and client primitives used by the platform |

The provider calls supported platform APIs. Cloud implementation details stay
behind the platform service boundary. Platform infrastructure continues to use
the AWS provider; customer Terraform declares a database or service rather than
the backing ECS/EFS/S3 resources. Keep one reconciler per mutable field.

Use a thin Go HTTP client when the platform exposes a public HTTP contract.
Generate boundary types from the owning published schema and validate responses.
If a selected interface requires native Transit signing or protocol semantics,
reuse the shared Zig client boundary. Do not duplicate that protocol in Go or
introduce a mandatory native dependency merely because Transit is written in Zig.

## Starting state and source evidence

The following were inspected while preparing this handoff; refresh them before
implementation and distinguish later changes from this source basis.

| Checkout | Inspected state |
| --- | --- |
| `~/workspace/spoke-sh/terraform-provider-spoke` | Clean `main` at `2f1affc4911ef72e8288222a77198942d8ceba9f` before this file; remote `git@github.com:spoke-sh/terraform-provider-spoke.git`; only `README.md` and MIT `LICENSE` tracked |
| `~/workspace/spoke-sh/platform` | Clean `main` at `78cebeb173916810c107d6a90ed5711f63f4d742` |
| `~/workspace/spoke-sh/transit` | Clean `main` at `0e2fba44727ca858bd1e8e4a775b62fbcf1a665d`, the merge of Transit PR #291 |

There was no `AGENTS.md` in the provider checkout or its parent directories at
inspection. Read any subsequently added instructions before editing. Read the
destination repository's own instructions for cross-repository work; Transit
runtime rules and platform task limits have their respective scopes.

No provider code, Go module, flake, API fixture or CI workflow exists in this
checkout yet. The old `transit-terraform-provider-plan` worktree no longer exists
at its earlier sibling path. Work from this provider repository.

### Existing tracker and corrections

The detailed original plan is
[Transit issue #290](https://github.com/spoke-sh/transit/issues/290). It remains
open and was not edited during this handoff. Retain its resource lifecycle,
uncertainty, Nix and acceptance requirements, with these corrections:

1. Provider source belongs here, not `transit/sdks/go/terraform-provider-spoke/`.
2. This repository is the actual source/release owner. No generated GitHub
   publishing mirror is needed.
3. Provider releases, dependency pins and Nix inputs are owned here. Downstream
   users pin this provider's flake/release, not a Transit revision to obtain it.
4. Platform PR #6 is now merged and uses Transit #291. The issue's earlier
   description of an open PR pinned to closed Transit #279 is historical.

Tracking follow-up: transfer or recreate the provider implementation tracker in
this repository with cross-links, keep platform API/service work in `platform`,
and retain generic upstream dependencies in `transit`. Preserve the original
discussion and replace stale location/version assumptions throughout the plan.

### Platform work to consume

- [Platform PR #6](https://github.com/spoke-sh/platform/pull/6) merged as
  `3560c31124b890d87dd8dc7e03461fc07d56824f`. It adopts
  [Transit PR #291](https://github.com/spoke-sh/transit/pull/291), native
  migrations and typed browser declarations, and prepares the private AWS
  foundation. The Node/Lambda API bridge was removed.
- [Platform PR #7](https://github.com/spoke-sh/platform/pull/7) merged as the
  inspected platform HEAD. Its store-and-activate design is product context,
  not evidence of executable service provisioning.
- [Current runtime handoff](https://github.com/spoke-sh/platform/blob/78cebeb173916810c107d6a90ed5711f63f4d742/docs/transit-compatibility.md)
  pins Transit `0e2fba44727ca858bd1e8e4a775b62fbcf1a665d`. Browser publication
  uses typed `transit-admin browser`; migrations use the native lifecycle.
  Qualification covers fresh installations and replacement with the same native
  definition, not an in-place upgrade of old #279 installations.
- [Transit #284](https://github.com/spoke-sh/transit/issues/284) still showed
  open during inspection despite the merged runtime implementation. Evaluate
  its acceptance items against #291 instead of treating its issue state as
  evidence that all browser integration is absent.
- [Transit #277](https://github.com/spoke-sh/transit/issues/277) still tracks
  runtime image accessibility. It is separate from provider distribution.

Read the platform's
[contracts](https://github.com/spoke-sh/platform/blob/78cebeb173916810c107d6a90ed5711f63f4d742/docs/platform-contracts.md),
[ownership matrix](https://github.com/spoke-sh/platform/blob/78cebeb173916810c107d6a90ed5711f63f4d742/docs/resource-ownership.md),
[roadmap](https://github.com/spoke-sh/platform/blob/78cebeb173916810c107d6a90ed5711f63f4d742/docs/roadmap/README.md)
and [qualification record](https://github.com/spoke-sh/platform/blob/78cebeb173916810c107d6a90ed5711f63f4d742/docs/qualification/phase-1.md).
Milestones 4–6 own artifacts/builds, isolated workloads and customer data;
7–9 cover scheduling, recovery and placement. These customer capabilities remain
planned in the inspected source. `platform_surfaces` rows describe navigation;
they are not provisioning requests or authorization records.

## Proposed resource inventory

Names below are candidates, not implemented Terraform schemas.

| Area | Proposed resources | Boundary |
| --- | --- | --- |
| Ownership | `spoke_project`, `spoke_environment` | Stable customer scope; an environment does not implicitly create an AWS account or Transit realm |
| Access | `spoke_service_account`, `spoke_grant` | Scoped machine identity and attributable grants; credential issuance/rotation is separate |
| Artifacts | `spoke_artifact_registry` | Private OCI registry/repository, access, quotas and retention; deploy immutable digests |
| Data | `spoke_database` | Managed Transit database service with explicit shared/dedicated profile, discovery, capacity and retention; SQL wire support does not imply a PostgreSQL/MySQL engine |
| Objects | `spoke_bucket` | Customer object storage with a defined data API, access and deletion policy; distinct from internal segments and Terraform state |
| Compute | `spoke_container_service` | Stable deployment identity, digest, resource limits, configuration, health and desired revision; instances are observations |
| Bindings | `spoke_secret_reference`, `spoke_service_binding` | Secret references and workload connections to services; binding does not implicitly grant access |
| Ingress | `spoke_endpoint` | Explicit private/public routing and TLS; custom domains need ownership verification |

Add read-only data sources for existing scoped services and artifact digests.
Later candidates include jobs/schedules, streams/queues, volumes/attachments,
custom domains, backup policies, database clones, private connectivity and
scaling policies. Each needs a backend lifecycle before provider exposure.

An optional `spoke_bucket_object` could manage bounded deployment files by
digest and exact object/version. Ordinary application files and large datasets
use the data API. Do not store file bytes in Terraform state or treat an object
bucket as a POSIX filesystem. Builds, individual job executions, database rows
and physical container instances should not become ordinary managed resources.

## First implementation sequence

1. **Audit the management boundary.** Build a capability matrix for proposed
   resources: public operation, authentication, stable ID, create/read/update/
   delete/import, revision checks, operation recovery, dependencies, deletion
   semantics and executable evidence. Record missing API contracts with their
   platform owners. Do not infer endpoints from UI mocks or private Tables.
2. **Scaffold this repository.** Use the Go Terraform Plugin Framework and
   protocol 6, with independently pinned dependencies. A reasonable layout is
   root `go.mod`/`go.sum` and `main.go`, `internal/provider/`, a small
   `internal/platformclient/`, `examples/`, generated `docs/`, `flake.nix` and
   CI workflows. Create only files needed by actual work. Preserve the MIT
   license and improve the minimal README as implementation lands.
3. **Establish a narrow real slice.** Prefer project/environment discovery and
   one managed resource once its API exists. Prove create, refresh, import,
   update, empty second plan and dependency-safe delete before broadening.
   Scaffolding, validation and client tests can progress while API work proceeds;
   do not register fake resources or report mocked provisioning as delivered.
4. **Extend in dependency order.** Scope/access → registry/database/bucket →
   container/bindings/endpoint. The complete product journey is build/import,
   deploy, grant database/bucket access, inspect, update, replace, revoke and
   retire while preserving separately owned data.
5. **Ship Nix and external-consumer checks.** Make another repository install
   the exact provider and exercise the real slice without source-tree imports,
   a compiler or development overrides.
6. **Publish evidence and tracking.** Keep provider docs current; update platform
   contracts where service semantics land and Transit owners where generic
   primitives change. Track remaining service prerequisites separately.

Platform currently requires Terraform `>= 1.10, < 2.0`. Select and test the
provider's minimum deliberately against its chosen Framework features. Keep
provider configuration explicit: endpoint, scope/target assertions, verified
TLS, external machine credentials and bounded operation timeouts. Browser
sign-in is not an automation credential or a provisioning grant.

## Lifecycle and recovery requirements

- Keep logical object ID, desired revision, accepted operation, active revision,
  health and physical incarnation separate. Acceptance alone is not completion.
- Plan, refresh and data-source reads are observational. They must not build,
  upload, run migrations, start jobs or reconcile mutations. Revalidate
  permissions and revision preconditions at apply.
- Establish recoverable request identity before the first mutation. A provider
  can die after server acceptance but before Terraform records its resource ID.
  Define stable management scope/key, request fingerprint, operation generations
  and authorized pending-operation discovery in the platform contract.
- A configuration hash alone cannot distinguish a retry from a later legitimate
  return to previous configuration. Provider private state is additional
  evidence, not a durable pre-dispatch journal. If a native interface needs
  saved signed envelopes, specify and qualify the recovery store.
- Timeouts/cancellation stop waiting; they do not prove rollback. Preserve and
  inspect the original accepted operation. Do not retry uncertain work with a
  fresh identity, or silently adopt an unrelated same-name object.
- Import uses stable scoped IDs. Only authoritative absence permits removing
  an object from state; denial, outage and incomplete pagination are errors.
- Reconcile concurrent changes with native revisions/fences. Bound request work,
  pages and settlement independently of total durable resource count.
- Default to dependency-restricted deletion and explicit data protection.
  Removing a service leaves separately owned databases, buckets and artifacts
  intact. No implicit purge, force-empty, cascade or retention policy changes.
- Credentials and secret values stay outside state, plans, diagnostics, logs
  and the Nix store. A Terraform sensitive flag alone does not prevent storage.
- Terraform, platform operations and cloud effects are not one transaction.
  Platform controllers preserve accepted intent and recover uncertain effects.

Schema/data migrations retain the native migration owner. Do not implement
provisioners that shell out to SQL, ECS Exec, AWS CLI or Nix as the provider's
service implementation. Do not mutate Board inspection surfaces or raw storage.

## Nix and distribution

Own a pinned flake here. Proposed outputs are
`packages.<system>.terraform-provider-spoke`, a versioned provider mirror and
`devShells.<system>.terraform`. Use `buildGoModule` with verified dependency
hashes and preserve generated schemas in source inputs.

Provide a consumer example with a pinned provider flake, project-scoped
`TF_CLI_CONFIG_FILE`, filesystem mirror and `.terraform.lock.hcl`. Exclude this
provider from direct registry lookup while it is mirror-only without preventing
other providers from installing. `dev_overrides` is for provider development;
it bypasses normal version/checksum selection and is not the release install path.
Nix evaluation, build and shell entry perform no deployment effects.

When mirror and registry share a version, distribute identical provider bytes.
Otherwise use a distinct immutable prerelease identity for source-built Nix
artifacts. Never disable lock checks or replace bytes under a published version.

Application delivery is a separate workflow:

```text
pinned source + flake.lock + output/system
  -> Nix build
  -> explicit authorized registry upload/promotion
  -> verified immutable OCI digest
  -> Terraform service deployment
```

A Nix store path is not a remotely deployable image. Hosted customer builds
need the platform's isolated evaluator/builder and promotion contracts. Nix
caches, OCI registries and application buckets have separate access policies.

This repository already has the Registry-compatible name
`terraform-provider-spoke`; no publishing mirror repository is necessary.
Add independent release automation for platform ZIPs, signed checksums, protocol
manifest and generated docs. Test Linux amd64 first, then Linux arm64 and macOS
on amd64/arm64 before advertising support. A remote Go HTTP provider need not
inherit local Transit runtime restrictions; native dependencies add their own
portability and ABI checks. Keep source provenance and API compatibility explicit.

## Validation and downstream adoption

Use Framework unit and acceptance tests plus real service fixtures. Required
cases include import, empty second plan, drift, rename/recreation, denied reads,
stale revisions, cross-tenant access, quota refusal, nonempty data protection,
dependency-restricted deletion and interrupted create/update/delete. Kill the
provider before its response, lose acceptance/completion replies, restart the
platform/controller and exercise concurrent applies. Verify recovery preserves
the original identity and never duplicates an active resource.

Test registry push/pull, bucket upload/download, workload data access and actual
endpoint behavior independently of Terraform state. Test anonymous provider
installation in a clean external consumer and the checked lockfile/mirror path.
Record mocks, component checks, real backend tests and cloud qualification
separately; skipped or incomplete checks establish no service guarantee.

In `platform`, keep `infra/bootstrap` and `infra/platform` as infrastructure
roots. A customer-service root/example must consume an already available
management endpoint; avoid requiring an uncreated platform to configure its own
provider. Preserve an independent recovery path for the foundation.

Platform's `AGENTS.md` permits at most seven public just recipes and requires
credential-free Terraform format/lint/validation checks without initializing S3
state. Real service acceptance belongs in an explicit isolated integration lane.
Those platform limits do not automatically define this repository's task layout.

## References and handoff completion

- [Original implementation tracker](https://github.com/spoke-sh/transit/issues/290)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Framework private state](https://developer.hashicorp.com/terraform/plugin/framework/resources/private-state)
- [Provider publishing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing)
- [Terraform CLI mirrors and development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file)
- [Nixpkgs Go packaging](https://nixos.org/manual/nixpkgs/unstable/#sec-language-go)

This handoff creates documentation only. It does not implement a provider,
register a Terraform namespace, change GitHub trackers or qualify any service.
The first next-agent deliverable should identify the actual API gaps and supply
a checked provider/Nix foundation, followed by a real supported resource slice.
