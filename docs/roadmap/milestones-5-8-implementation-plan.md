# Milestones 5–8 implementation plan

[Roadmap](README.md) · **Plan status:** In Progress · **Review:** Authorized for implementation

Extend the completed scope/access foundation with registries, databases and
expanded buckets, secret references and bindings, then container services and
endpoints. Qualify the complete application graph through real Terraform and
mocked platform management endpoints. This is a proposal for review; implementation
of milestones 5–8 follows in subsequent changes. The numbered milestone files
remain the owners of scope, acceptance criteria and completion status.

## Decisions for review

These are recommended implementation choices, not published platform guarantees.
Review them together with the milestone exit gates before starting resource work.
Record any accepted changes in this plan and the provisional contract.

| Decision | Proposed approach | Consequence to review |
| --- | --- | --- |
| Contract evolution | Introduce mock-v3 for typed service specifications; retain v2 fixtures and qualify explicit migration. | Existing `/v2` grant target paths need deliberate migration. Pending v2 work must finish or recover with the old provider before cutover. |
| Registry granularity | One environment-scoped registry resource; repositories and immutable digests are discovered metadata. | No repository resource or Terraform build/push step in milestone 5. |
| Database changes | Shared/dedicated profiles; profile changes require protected replacement. Initially permit capacity growth and refuse shrink. | Capacity reduction needs a later supported data-preserving contract; replacement cannot delete or adopt another data service. |
| Runtime authorization | Explicit service-specific grant actions, separate from management read/manage. | Creating a service or binding never authorizes data access; examples must declare grants for the runtime principal. |
| Secret rotation | External owner/identifier plus a pinned version; rotation is an explicit reference update. | No secret values or automatic latest-version tracking in Terraform. Consumer activation is observed separately. |
| Binding scope | Same-environment exact consumer/target identities with one owner per connection slot. | Cross-environment connections are excluded initially. Detach must precede deletion and preserve independent targets/grants. |
| Workload readiness | Complete resource creation when its management declaration settles; expose pending/failed/healthy rollout separately. | Terraform can create bindings after obtaining the consumer ID. Successful apply alone does not establish healthy workload traffic. |
| Endpoint exposure | Require explicit private/public selection and platform-managed TLS policy. | No accidental public default, certificate private keys, custom domains or private-network provisioning. |

## Proposed Terraform surface

All new resources use the established management key, scoped identity, principal
ownership, native revision, canonical import and recoverable operation rules.
The attributes below describe the minimum intended surface; wire names, units,
defaults and bounds must be fixed in the reviewed test schema before adapters.
Data-bearing services and registries retain protection by default and refuse
nonempty/in-use deletion even after protection is disabled.

| Milestone | Resource or data source | Configuration and observations |
| --- | --- | --- |
| 5 | `spoke_artifact_registry` | Name, OCI format, quota, retention and protection; endpoint and readiness metadata. |
| 5 | `data.spoke_artifact` | Exact registry/repository and one digest or tag selector; resolved immutable digest and metadata. |
| 6 | `spoke_database` | Name, shared/dedicated profile, capacity, retention and protection; protocol/endpoint and readiness without credentials. |
| 6 | Existing `spoke_bucket` | Add optional quota/capacity and retention settings plus endpoint metadata; preserve current IDs, keys and protection. |
| 7 | `spoke_secret_reference` | External owner, identifier and pinned version; observed resolution/configuration status without secret payloads. |
| 7 | `spoke_service_binding` | Exact consumer/target, connection slot and intent; observed effective configuration revision and status. |
| 8 | `spoke_container_service` | Registry/repository/digest, service account, resource bounds, non-secret configuration and secret references; desired/effective/active revisions and health. |
| 8 | `spoke_endpoint` | Exact service/port, explicit exposure and TLS policy; safe endpoint, routing and TLS observations. |

Add exact-ID and complete bounded collection data sources for each new managed
kind. Preserve the existing bucket collection and add an exact bucket lookup as
part of milestone 6. These reads never provision services or advance controllers.

## Baseline and boundaries

Start from merged revision
[`52e34df`](https://github.com/spoke-sh/terraform-provider-spoke/commit/52e34df),
which includes the implementation in
[PR #4](https://github.com/spoke-sh/terraform-provider-spoke/pull/4) and the roadmap
move/initial plan in [PR #5](https://github.com/spoke-sh/terraform-provider-spoke/pull/5).
[Milestone 4](04-scope-and-access.md) is **Completed**, with executed evidence in
the [milestones 2–4 qualification record](../../qualification/milestones-2-4.md).
Milestones 5–8 are **In Progress**. Milestone 4 is a
delivered dependency; its regression requirements are retained below.

The baseline provides five resources, seven data sources, mock-v2 negotiation,
bounded discovery, principal-bound receipts, revision fences, protected deletion
and installed-provider crash tests with persisted fixture reload. Reuse those
mechanisms. The current bucket-shaped wire spec is too narrow for the new service
settings; separate typed service specifications from the shared operation envelope
before extending it.

The [capability matrix](../../testdata/contracts/capabilities.md) records the
platform assessment at `3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740` on 2026-10-31.
It does not establish executable customer provisioning APIs. Refresh that source
assessment before implementing each service contract. Record published schemas
and upstream prerequisites there; until available, use explicitly provisional
fixtures. Platform owns provisioning, policy and reconciliation. Transit owns
generic runtime primitives. This repository owns the provider/client adapters
and test service, with no private-table or cloud-CLI provisioning shortcuts.

This phase does not implement an OCI registry, SQL/object data plane, secret
store, container runtime or ingress proxy in the fixture. Real service checks
belong to [milestone 10](10-live-platform-qualification.md). Nix distribution and
release qualification remain in [9](09-nix-packages-and-consumers.md) and
[11](11-compatibility-and-releases.md). Bucket objects are deferred to a separate
milestone if selected; custom domains/private networking remain in
[15](15-domains-and-private-connectivity.md).

## Dependencies and delivery order

| Step | Prerequisites | Reviewable outcome |
| --- | --- | --- |
| Shared contract extension | 2–4 | Versioned service specs, runtime access actions, readiness and acyclic binding rules before resource adapters. |
| 5 registries | Shared contracts and 4 | Registry lifecycle plus exact immutable digest discovery. |
| 6 data services | Shared contracts and 4 | Database lifecycle and expanded buckets. Registry implementation is not a technical prerequisite. |
| 7 references/bindings | 5–6 service identities; agreed 8 consumer contract | Reference and binding lifecycles against seeded consumers; no requirement for the container Terraform resource yet. |
| 8 workloads/endpoints | 5–7 | Container and endpoint lifecycles followed by the complete installed-provider graph. |

Implement 5 then 6 for small sequential reviews, while keeping them independently
testable. Agree the 7/8 dependency contract during shared design. Each service
lands contract/fixture/client work, its Terraform adapter and focused acceptance,
then examples/reference/evidence. Keep every commit buildable; do not register
unfinished resource kinds as supported capabilities.

## Shared contract and fixture work

### Versioned management boundary

The recommended contract change is provisional **mock-v3** for the typed service
envelope and new kinds. Confirm this decision during plan review before changing
routes or serialization. Preserve mock-v1/v2 specifications as historical baselines.
Document the intentional API-version transition; reject unsupported versions
before any mutation. Do not silently reinterpret a v2 receipt or fingerprint as
v3. The existing Terraform state identities and attributes remain stable; the
mock protocol version is separate from a Terraform resource schema version.

Use the existing environment scope as the proposed parent for all new services.
Under `/v3/organizations/{org}/projects/{project}/environments/{env}`, specify
collections `artifact_registries`, `databases`, `buckets`, `secret_references`,
`service_bindings`, `container_services` and `endpoints`. Retain organization
projects/service accounts/grants and project environments. These are proposed
test routes, not published platform endpoints. Define exact read/list/create/
update/delete, receipt and operation routes in the schema before adding handlers.
Digest metadata reads are nested under the exact registry and repository.

For every kind, record required/optional/computed fields, units and bounds,
null/unknown handling, immutable choices, valid updates, canonical import ID,
replacement semantics, readiness, error codes, quotas and deletion restrictions.
New environment-scoped import IDs follow `org/project/environment/resource`;
resource type supplies the collection. Digests additionally identify the exact
registry/repository. Names and mutable tags never become adoption keys.

Replace the broad `BucketSpec` reuse with typed specs and observations keyed by
resource kind. Share scope, ownership, revision, intent fingerprint, generation,
receipt and operation handling. Strictly reject missing/null required fields,
unknown fields, wrong kinds and scope mismatches. Define deterministic fingerprint
encoding for nested configuration, explicit defaults, lists and maps; test
semantically equivalent input ordering and changed intent separately.

Keep legacy Terraform state fixtures for all five existing resources. A protocol
upgrade must not schedule replacement or lose protection, ownership or recovery
keys. Because `target_path` currently includes `/v2`, explicitly migrate computed
targets and configured grant references to v3. Grant `target` currently requires
replacement on change: a simple configuration rewrite is insufficient. Coordinate
fixture and Terraform state migration so the same logical grant retains its ID,
owner and permission without revoke/recreate. Prove an empty plan after migration
with both resource-derived and literal grant targets, and reject unmigrated paths
rather than granting a different target. If this cannot be demonstrated, revise
the versioning proposal before adopting v3.
Pin old fixtures to their old contract; no cross-version receipt migration is
implied. Settle or recover outstanding v2 work using the pinned v2 provider and
fixture before migration; explicitly refuse an upgrade with pending old-version
work. Test completed-resource migration with a deliberate fixture migration
that retains object IDs and principal ownership, and test rejection when work is
still pending. Add Terraform state upgraders only where the persisted schema
changes, and test them against captured prior state before adding new settings.

### Policy, dependencies and controller observations

Extend explicit grants with service-specific runtime actions for artifact pull/
push, data access, secret resolution and endpoint invocation. Fix their names and
target restrictions in the contract/capability matrix. Management `read`/`manage`
must not silently grant data-plane access. Workloads use an explicit service
account identity whose credentials are issued externally. Verify authorization
through independent principals and exact targets; inspect runtime access using
test controls, not a new provider-side authorization implementation.

Persist accepted intents, dependency reservations and controller observations
before acknowledging transitions. Quota/dependency checks must account for both
existing objects and pending creates/updates at admission and settlement. Keep
ledger restart support for all new kinds. A delete racing a new binding, or an
update racing a retention change, must have a deterministic conflict outcome.

GET remains observational. Test controls explicitly settle management work,
publish synthetic artifact metadata, mark data present, rotate external secret
metadata, and advance rollout/TLS status. Separate management operation outcome,
desired configuration revision, active revision and readiness/health. A timeout
does not roll back accepted work or erase its receipt.

### Acyclic binding and workload contract

A container resource declares its image, identity and configuration; it does not
take IDs of bindings that themselves need its ID. A binding declares the exact
consumer, target and connection slot. The platform derives the consumer's
effective configuration from those independent declarations. A slot has one
owner, and conflicting bindings fail rather than overwrite one another.

Creating a logical container settles after its declaration is durably accepted
and management reconciliation completes. It may still report pending workload
readiness while bindings are being created. Do not require healthy workload
startup to return its ID: that would prevent Terraform from creating the binding
needed for startup. Derived binding/secret changes advance the effective
configuration observation without rewriting the container's Terraform-owned
image/configuration fields. Health failures remain visible observations with
reason codes; declaration failures remain operation errors.

Bindings depend on consumers and targets; endpoints depend on consumers and the
required bindings/grants. Terraform can therefore destroy endpoints, detach
bindings, then retire compute. Bound targets and consumers refuse premature
deletion. Use synthetic consumer records in milestone 7 tests and the real
container resource in milestone 8. Test initial deployment, later binding changes
and teardown through Terraform without manual state surgery or `-target`.

## Delivered foundation

Keep [the completed milestone](04-scope-and-access.md) as a regression gate.
Organization-only bootstrap creates projects/environments and service accounts;
a subsequent consumer uses the concrete scope IDs. Unknown provider target IDs
remain invalid. The end-to-end example has separate bootstrap and service roots
to avoid provider configuration cycles.

Extend grant validation and dependencies only as new kinds/actions arrive.
Preserve exact-target authority without sibling discovery, self-revocation
recovery after external restoration, independent grants, parent restrictions
and the existing canonical import identities. Do not issue credentials or add
implicit grants from service creation, binding or successful login. Run the
existing alias, saved-plan revision, bootstrap and independent-state scenarios
against the new contract alongside new service tests.

## Milestone 5: artifact registries and digest discovery

1. Specify `spoke_artifact_registry` with environment identity, OCI format,
   quota/retention policy, protection and computed endpoint/readiness metadata.
   Define registry versus repository identity, allowed policy changes and how
   retained/in-use content restricts update and deletion before implementation.
2. Extend typed client and fixture handling. Synthetic metadata models repository,
   digest, tags, retained content, quota use and consumer references. Explicit
   test controls publish artifacts; Terraform never uploads content. Add strict
   raw-HTTP tests independent of client serialization.
3. Add the resource, exact-ID registry reads, complete scoped collection discovery
   and `data.spoke_artifact` for an exact registry/repository plus digest or tag
   selector. Reject missing/ambiguous selectors, return an immutable digest, and
   distinguish absent metadata, denied access and incomplete results. All reads
   use the existing body/page/retry/cancellation bounds.
4. Exercise CRUD/import, no-change second plan, policy drift, stale revisions,
   quota/retention restrictions, interrupted mutations and owner conflicts. A
   changed tag can change a future lookup result; it cannot mutate an already
   pinned workload without a Terraform configuration/apply change.
5. Add examples that build from pinned inputs and explicitly push/promote outside
   Terraform before looking up/passing a digest. Nix store paths are local paths,
   not image identities. Keep registry credentials outside resource attributes.

Exit: [milestone 5 acceptance](05-artifact-registries.md#acceptance-criteria),
including retained/in-use deletion refusal and observable read-only discovery,
has executed mock evidence. OCI push/pull still requires milestone 10.

## Milestone 6: databases and expanded buckets

1. Define `spoke_database` profiles (shared/dedicated), capacity units/bounds,
   retention, protection, readiness and non-secret endpoint/protocol metadata.
   Initially allow capacity growth in place, reject shrink, and require protected
   replacement for a profile change. Record retention restrictions and service
   limits in the contract before schema implementation.
   SQL wire support alone does not establish PostgreSQL/MySQL compatibility.
2. Implement the schema, typed client, fixture and Terraform lifecycle with exact
   reads and bounded collection discovery. Model data presence and active
   dependencies as durable test observations; do not implement database queries
   or migrations in provider/fixture lifecycle callbacks.
3. Expand `spoke_bucket` with contract-backed capacity/quota, retention and
   endpoint metadata. Access stays in explicit grants. Prefer additive optional
   fields with specified defaults and retain current import IDs/management keys.
   Prove a saved baseline state refreshes to an empty plan without replacement;
   provide an upgrader if the stored representation must change.
4. Test allowed growth without identity loss, rejected capacity reduction,
   retention changes, protection, nonempty/in-use deletion, quota reservation and
   concurrent updates. Replacement must respect protection and leave the old
   data service intact on failed creation; never silently adopt or purge it.
   A nonempty database cannot be replaced in place. Moving its data requires a
   separately declared database with a new management key and an external data
   migration; Terraform does not copy rows or bypass the old service's protection.
5. Document separate database management versus schema/row operations and object
   service management versus file transfer. Keep `spoke_bucket_object` deferred;
   no file bytes, database passwords or credential-bearing URLs enter state.

Exit: [milestone 6 acceptance](06-databases-and-object-storage.md#acceptance-criteria)
and bucket state compatibility pass. Actual database access and object
upload/download remain live qualification items.

## Milestone 7: secret references and bindings

1. Implement `spoke_secret_reference` around external owner, identifier and pinned
   version metadata. Define registration versus resolution authority, version
   change semantics and observed resolved version. Initially require an explicit
   version; rotation is an intentional reference update. Do not fetch or return
   secret bytes, including through API error bodies.
2. Implement `spoke_service_binding` with exact consumer/target identities,
   connection slot/intent, immutable relationship fields and an observed
   configuration revision/status. Replacement detaches only the owned binding.
   Define permitted target kinds and reject cross-scope relationships up front;
   do not infer another environment from a bare ID.
3. Add exact reads and complete discovery for both kinds. Seed consumers through
   test-only controls. Model missing targets, conflicting slots, denied resolution,
   rotated versions, deleted consumers and revoked access without secret content.
   A declaration never creates a grant; absent authority produces the documented
   denial or unresolved observation and can never imply usable access.
4. Exercise lifecycle/import/drift, interrupted create/update/delete, revision
   conflicts, independent owners and retained dependencies. Deleting a bound
   service/reference or consumer fails until detach; detach preserves the target,
   external secret and every independent grant. Losing inspection rights keeps
   known Terraform state until authority is restored.
5. Inject synthetic secret canaries into external fixture metadata/error controls
   and inspect state, saved plans, stdout/stderr, diagnostics and provider logs
   through success, rotation, denial and timeout. Only identifiers/versions and
   safe status codes may be returned. Show explicit runtime grants in examples.

Exit: [milestone 7 acceptance](07-secret-references-and-bindings.md#acceptance-criteria)
passes against seeded consumers. Full Terraform workload composition follows in
8; real secret resolution/access remains in 10.

## Milestone 8: containers, endpoints and the complete graph

1. Implement `spoke_container_service` with registry/repository and immutable
   digest, explicit service account, bounded resource requirements, non-secret
   configuration and secret-reference metadata. Validate digests and settings
   before mutation. Keep logical service identity stable across allowed updates;
   physical instances and individual builds are observations/external work.
2. Expose management operation, desired revision, effective configuration
   revision, active revision and rollout/health separately. Add controller-driven
   fixtures for missing images, denied pull, unavailable bindings, failed health,
   retained old active revision and successful digest replacement. Polling cannot
   advance any transition. Bounded mutation waiting retains original recovery
   identity on cancellation/timeout; it cannot claim rollback or healthy rollout.
3. Implement `spoke_endpoint` with exact service/port, explicitly selected
   private/public exposure and supported TLS policy. Return safe endpoint and TLS
   observations; never accept private keys or silently default to public access.
   Define pending/failed certificate or target readiness independently of durable
   declaration completion. Exclude custom domains and private connectivity.
4. Add exact reads, bounded collection discovery, import, drift, revision and
   interrupted CRUD tests for both resources. Verify that updating a digest
   changes desired intent once, preserves ownership and eventually observes the
   independently advanced active revision. An endpoint's existence alone never
   proves successful application traffic.
5. Add an installed-provider service consumer rooted in concrete bootstrap
   outputs. Create identity/grants, registry and synthetic uploaded digest,
   database, bucket, secret reference, service, bindings and endpoint. Assert
   complete topology and explicit runtime authority using independent fixture
   observations. Update digest and secret version, revoke/restore runtime access,
   then retire endpoint/bindings/compute while retaining separately owned data,
   artifacts, secret metadata and grants.
6. Interrupt the partially applied graph after durable service acceptance and
   after binding settlement, restart the provider and fixture ledger, then apply
   again from the consumer state. Assert one logical object/effect per owned
   management key, preserved dependency reservations and an empty final plan.
   Use fixture-controlled transitions/barriers, not timing-dependent sleeps.

Exit: [milestone 8 acceptance](08-container-services-and-endpoints.md#acceptance-criteria)
and the cross-resource journey pass through the installed binary. Record that
the fixture models management metadata, not actual container execution/traffic.

## Implementation owners

| Location | Planned changes |
| --- | --- |
| `testdata/contracts/` | mock-v3 wire schema, raw request/response fixtures, version/migration rules and updated service capability matrix. Preserve earlier contract evidence. |
| `internal/platformclient/` | Typed per-service specifications/observations, exact/digest reads, bounded discovery, runtime permission types and shared revision/receipt handling. Keep existing recovery helpers and add service-specific cases. |
| `internal/testutil/mockplatform/` | Environment-scoped service handlers, durable dependency/quota reservations and synthetic artifact/data/secret/rollout metadata. Add explicit controller/fault controls and reload coverage. |
| `internal/provider/` | Explicit resource and data-source schemas for each slice, additive bucket settings, required state upgraders and registration only after each kind is testable. Extend lifecycle and installed-process acceptance. |
| `examples/` and generated `docs/` reference | Individual resources/imports/data sources, separate bootstrap/service consumers and a complete service-graph example with explicit grants. Regenerate through `scripts/generate-docs.sh`. |
| `qualification/` and `docs/roadmap/` | Executed evidence per milestone, exact tested revisions/versions and synchronized milestone/index statuses after all exit gates pass. |

## Test and evidence matrix

| Layer | Required evidence |
| --- | --- |
| Contract/raw HTTP | Independent JSON fixtures per kind, strict schema validation, version/principal/scope checks, immutable identities, allowed mutations, runtime permission boundaries and each service's error/dependency rules. |
| Client | Verified TLS, bounded bodies/pages/retries/waits, cancellation, incomplete/denied discovery, fingerprint/generation/revision conflicts, receipt expiry/tombstones and no blind retry after ambiguous acceptance. |
| Per-resource Terraform | Known/unknown/null validation, create/read/update/delete, exact import and owner checks, empty second plan, permitted versus replacement changes, drift, authoritative absence, denied refresh preserving state and protected/in-use deletion. |
| Recovery | Every new mutable kind covers CRUD across pre-dispatch, pre-acceptance, accepted/lost-reply and completed/lost-state barriers with fixture reload. Extend the installed-process harness for service-specific pending/readiness/dependency state, not only the original bucket. |
| Cross-resource policy/data | Concurrent reservations, explicit grants, no binding-induced authorization, self-revocation/restoration, independent states/owners/aliases, safe detach/retirement and retained unrelated data/artifacts/grants. |
| Secrets | Synthetic credential/secret canaries absent from Terraform state, saved plans, diagnostics and logs across success, rotation and failure. Reference schemas accept identifiers/versions and reject secret-payload fields. |
| Compatibility | Captured v2-era Terraform states retain identities/protection on the intended v3 migration; existing grant target-path changes are explicit; no unsupported receipt translation or silent state adoption. |
| Complete graph | Installed checksum-locked provider, separate bootstrap/consumer, no Go on consumer PATH, partially applied graph recovery, digest/version update, revocation and selective retirement. |

Expand `internal/platformclient/` for service wire types and shared recovery,
`internal/testutil/mockplatform/` for durable metadata/controller controls, and
`internal/provider/` for explicit schemas/adapters and acceptance. Reuse lifecycle
helpers only where behavior is actually shared; keep per-kind constraints visible.
Extend the existing installed-process harness for aliases and crashes because
in-process provider reattachment shares configuration across aliases.

Before handing off each implementation slice, run `just fmt`, `just check` and
`just build` inside the pinned Nix shell. Ordinary CI must run all required mock
cases with synthetic credentials and fail if they are missing/skipped. Keep five
public Just tasks and the default task listing. Live mode requires explicit
selection/configuration and must not fall back to the fixture.

After schema/example edits, run
`nix develop --command ./scripts/generate-docs.sh`; verify `docs/roadmap/` survives
and the generated reference matches the implemented experimental surface.
Update the root README, capability matrix and examples with each slice. Add
qualification records under `qualification/` with exact committed revision,
commands/results, Terraform/Go/system versions, fixture/API version and limits.
Only then update the milestone and index together to `Completed`. Mock process
restart evidence is not a real service durability guarantee.

## Suggested logical commits

| Commit group | Contents and gate |
| --- | --- |
| 1. Service contracts | mock-v3 schema/design, typed envelope/fingerprints, runtime permission and dependency contracts, old-state/target-path migration policy; raw contract and baseline regression tests. |
| 2–3. Registry slice | Client/fixture/digest reads, then resource/data sources, Terraform acceptance, examples/reference and milestone 5 evidence. |
| 4–6. Data slice | Database client/fixture/adapter; bucket settings and state compatibility; data-protection/recovery evidence and milestone 6 documentation. |
| 7–8. Reference/binding slice | External reference metadata and canary tests; seeded-consumer bindings, explicit access/detach tests, examples/reference and milestone 7 evidence. |
| 9–10. Workload/endpoint slice | Container reconciliation observations, then explicit routing/TLS resources with lifecycle/import/recovery tests and generated reference. |
| 11. Graph qualification | Installed-consumer journey, partial-apply recovery, selective retirement, README/examples and executed milestone 8 evidence. |

Keep milestone 5 and milestone 6 in independently reviewable PRs after the shared
contract changes; then deliver milestone 7 and milestone 8 in dependency order.
Every PR includes its own tests and reference/examples. The final graph PR records
combined evidence without substituting for earlier per-resource gates.

Split a group further when needed to keep reviews focused and tests green.
Refresh PR descriptions around the delivered scope and cite executed checks.
Open platform prerequisites only with verified source context; tracking work is
not backend agreement. No milestone is completed merely by merging its schema
or this plan.

## Review checklist

- Confirm the proposed resource inventory and exclusions for milestones 5–8.
- Review the mock-v3 cutover and required state/grant-path compatibility evidence.
- Confirm explicit runtime grants, pinned secret versions, capacity/profile
  change rules and same-environment binding restrictions.
- Confirm the distinction between a settled declaration and healthy runtime,
  including how binding creation and endpoint teardown avoid dependency cycles.
- Review the commit/PR sequence and mock, installed-provider and eventual live
  qualification gates. Milestone statuses change only with executed evidence.
