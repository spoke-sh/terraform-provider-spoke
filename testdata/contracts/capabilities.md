# Management capability and ownership matrix

Historical assessment for milestones 5–8: provider implementation starts at `a97dfca`; platform checkout
`5c170b77065981c079ed51e24c22026e341196be`, inspected 2026-10-31. Platform's
[contract baseline](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/platform-contracts.md)
explicitly says customer organization/provisioning APIs are not executable.
Browser publication, private tables and prepared AWS Terraform are not substitutes.

A subsequent source review at platform
`b13c86dbb83c1a40dd65884e35b9c6afa10ef6a4` found a locally qualified native
bucket-metadata profile with signed commands and durable receipts. The
[9–11 plan](../../docs/roadmap/milestones-9-11-implementation-plan.md#baseline-and-refreshed-upstream-assessment)
records this newer boundary and its SDK/runtime prerequisites. The previous
assessment and rows below remain the historical basis of the mock-v3 qualification; they do
not establish compatibility with the native profile or real service data planes.

All rows use external automation credentials, explicit tenant scope, immutable
IDs, canonical import identities, revision/generation preconditions, asynchronous
settlement and recoverable owner-bound receipts. Those are **draft requirements**,
not verified live API behavior. The [mock-v4 contract](README.md) defines concrete
routes and errors for implemented fixture kinds. Provider/mock execution evidence
is recorded historically in the [milestones 5–8 qualification](../../qualification/milestones-5-8.md)
and currently in the [mock-v4 / Nix record](../../qualification/milestones-9-11.md).
Every live-service claim requires a published platform schema covering those
operations plus milestone 10's installed/live evidence; the restricted native
metadata profile does not satisfy the full service rows.

| Resource / capability | Parent scope and draft operations | Deletion / dependencies | Fixture/provider owner | Live prerequisite and platform owner |
| --- | --- | --- | --- | --- |
| Target and capabilities | GET scope/capabilities; explicit version/principal/kinds | No mutation | Milestone 2 | Published management target/version negotiation and machine auth; platform contracts and scope/access milestone 5 |
| Project | Organization projects: CRUD, exact/list reads, import | Protection; environments/grants/pending children restrict delete | Milestones 2–4 | Stable customer project/tenancy API; platform scope/access milestone 5 and data milestone 6 |
| Environment | Project environments: CRUD, exact/list reads, import | Protection; services/grants/pending children restrict delete | Milestones 2–4 | Explicit environment lifecycle and isolation; platform milestones 5/11/6 |
| Service account | Organization service_accounts: CRUD, exact/list reads, import | Protection; dependent grants/consumers restrict delete; issuance remains external | Milestone 4 | Automation principal lifecycle/credential policy; platform scope/access milestone 5 |
| Grant | Organization grants: CRUD, exact/list reads, import | Exact owned grant only; no implicit grant from binding | Milestone 4 | Product permission adapter, revocation/freshness, attribution; platform scope/data milestones 5/6 |
| Bucket | Environment buckets: CRUD, exact/list reads, import | Protection, nonempty and dependent grants; no force empty | Foundation; expanded service settings milestone 6 | Customer object service/data API, retention/quota; platform data milestone 6 |
| Artifact registry / digest reads | Environment-scoped typed registry lifecycle and exact digest metadata | Retained/in-use artifacts and explicit retention | Milestone 5; mock-v4 fixture and Terraform adapter implemented | Registry lifecycle, OCI push/pull and immutable digest policy; platform artifacts milestone 10 |
| Database / reads | Environment-scoped database lifecycle; shared/dedicated, bounded capacity/retention | Data protection, retention, consumers | Milestone 6; mock-v4 fixture and Terraform adapter implemented | Managed database profile/readiness/data API; platform data milestone 6 |
| Secret reference | Environment-scoped external owner/identifier/pinned version metadata | Remove reference only; preserve secret/consumers | Milestone 7; mock-v4 fixture and Terraform adapter implemented | Resolution authority/rotation contract; platform workload/data milestones 11/6 |
| Service binding | Same-environment exact consumer/target/slot/intent | Explicit detach; no implicit authorization/cascade | Milestone 7; mock-v4 fixture and Terraform adapter implemented | Dependency graph and connection contract; platform workload/data milestones 11/6 |
| Container service | Digest/configuration intent with independent desired/effective/active observations | Retire compute; retain independently owned services | Milestone 8; mock-v4 fixture and Terraform adapter implemented | Accepted deployment, active revision, health and recovery; platform workloads milestone 11 |
| Endpoint | Exact service/port, explicit private/public and managed TLS | Remove route, preserve target/data; custom domains later | Milestone 8; mock-v4 fixture and Terraform adapter implemented | Public/private ingress policy, TLS and reachability; platform workloads milestone 11 |
| Operation / management receipt | Scoped GET; owner/key/generation/fingerprint; tombstone on expiry | Pending cannot expire; transfer unsupported | Milestones 2–3 | Durable acceptance, fingerprint encoding, independent owner identity, retention/transfer and real effect recovery; platform management milestone 4 and recovery milestone 18 |

Upstream owners are the platform's
[identity](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/roadmap/05-customer-scope-and-access.md),
[artifacts](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/roadmap/10-artifact-registries-and-isolated-builds.md),
[workloads](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/roadmap/11-workload-isolation-bindings-and-endpoints.md),
[data](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/roadmap/06-managed-databases-and-object-storage.md)
and [recovery](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/roadmap/18-durability-recovery-and-upgrades.md)
milestones. No cross-repository implementation or agreement is implied.

## Decisions still requiring platform agreement

- Automation auth, target selection and scope hierarchy, including parent bootstrap.
- Capability/version negotiation and published schema/type-generation ownership.
- Opaque identifiers, authoritative absence versus denial, and list completeness.
- Native revision conditions, admission/quota reservations and settlement failures.
- Request fingerprint format, durable owner identity, receipt retention/tombstones,
  explicit ownership transfer and handling of credential-cloned Terraform states.
- Per-service data protection, dependency graph and readiness/data-plane guarantees.

A supported contract must resolve these with its owner. Adopt its schema, update
client/fixtures/provider together and rerun applicable contract, recovery and live
scenarios. Never weaken a mock merely to hide a live divergence.

## Current native integration gate

The assessment at platform `230c2c4cee12c08aaeaaf65e3c252a6d50450a55` confirms the
same restricted native profile. Application codecs pass independent vectors, but
the shared Go SDK cannot yet prepare Serving commands or required subject proofs.
No native adapter, durable signed-work recovery, or real service data plane is
qualified. The [boundary record](../../qualification/native-boundary.md) identifies
exact owners and missing interfaces. The explicit live lane produces blocked
machine-readable evidence; it never treats mock-v4 as the native API.
