# Management capability and ownership matrix

Source assessment: provider implementation starts at `a97dfca`; platform checkout
`5c170b77065981c079ed51e24c22026e341196be`, inspected 2026-10-31. Platform's
[contract baseline](https://github.com/spoke-sh/platform/blob/5c170b77065981c079ed51e24c22026e341196be/docs/platform-contracts.md)
explicitly says customer organization/provisioning APIs are not executable.
Browser publication, private tables and prepared AWS Terraform are not substitutes.

All rows use external automation credentials, explicit tenant scope, immutable
IDs, canonical import identities, revision/generation preconditions, asynchronous
settlement and recoverable owner-bound receipts. Those are **draft requirements**,
not verified live API behavior. The [mock-v3 contract](README.md) defines concrete
routes and errors for implemented fixture kinds. Every live row is blocked on a
published platform schema plus milestone 10's installed/live evidence.

| Resource / capability | Parent scope and draft operations | Deletion / dependencies | Fixture/provider owner | Live prerequisite and platform owner |
| --- | --- | --- | --- | --- |
| Target and capabilities | GET scope/capabilities; explicit version/principal/kinds | No mutation | Milestone 2 | Published management target/version negotiation and machine auth; platform contracts and scope/access milestone 5 |
| Project | Organization projects: CRUD, exact/list reads, import | Protection; environments/grants/pending children restrict delete | Milestones 2–4 | Stable customer project/tenancy API; platform scope/access milestone 5 and data milestone 6 |
| Environment | Project environments: CRUD, exact/list reads, import | Protection; buckets/grants/pending children restrict delete | Milestones 2–4 | Explicit environment lifecycle and isolation; platform milestones 5/11/6 |
| Service account | Organization service_accounts: CRUD, exact/list reads, import | Protection; dependent grants restrict delete; issuance remains external | Milestone 4 | Automation principal lifecycle/credential policy; platform scope/access milestone 5 |
| Grant | Organization grants: CRUD, exact/list reads, import | Exact owned grant only; no implicit grant from binding | Milestone 4 | Product permission adapter, revocation/freshness, attribution; platform scope/data milestones 5/6 |
| Bucket | Environment buckets: CRUD, exact/list reads, import | Protection, nonempty and dependent grants; no force empty | Foundation; expanded service settings milestone 6 | Customer object service/data API, retention/quota; platform data milestone 6 |
| Artifact registry / digest reads | Environment-scoped typed registry lifecycle and exact digest metadata | Retained/in-use artifacts and explicit retention | Milestone 5; mock-v3 fixture implemented; Terraform adapter in progress | Registry lifecycle, OCI push/pull and immutable digest policy; platform artifacts milestone 10 |
| Database / reads | Environment-scoped database lifecycle; shared/dedicated, bounded capacity/retention | Data protection, retention, consumers | Milestone 6; mock-v3 fixture implemented; Terraform adapter in progress | Managed database profile/readiness/data API; platform data milestone 6 |
| Secret reference | Environment-scoped external owner/identifier/pinned version metadata | Remove reference only; preserve secret/consumers | Milestone 7; mock-v3 fixture implemented; Terraform adapter in progress | Resolution authority/rotation contract; platform workload/data milestones 11/6 |
| Service binding | Same-environment exact consumer/target/slot/intent | Explicit detach; no implicit authorization/cascade | Milestone 7; mock-v3 fixture implemented; Terraform adapter in progress | Dependency graph and connection contract; platform workload/data milestones 11/6 |
| Container service | Digest/configuration intent with independent desired/effective/active observations | Retire compute; retain independently owned services | Milestone 8; mock-v3 fixture implemented; Terraform adapter in progress | Accepted deployment, active revision, health and recovery; platform workloads milestone 11 |
| Endpoint | Exact service/port, explicit private/public and managed TLS | Remove route, preserve target/data; custom domains later | Milestone 8; mock-v3 fixture implemented; Terraform adapter in progress | Public/private ingress policy, TLS and reachability; platform workloads milestone 11 |
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
