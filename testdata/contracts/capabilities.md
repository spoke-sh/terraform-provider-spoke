# Management capability and ownership matrix

Source assessment: provider work starts at `d461004`; platform checkout
`3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740`, inspected 2026-10-31. Platform's
[contract baseline](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/platform-contracts.md)
explicitly says customer organization/provisioning APIs are not executable.
Browser publication, private tables and prepared AWS Terraform are not substitutes.

All rows use external automation credentials, explicit tenant scope, immutable
IDs, canonical import identities, revision/generation preconditions, asynchronous
settlement and recoverable owner-bound receipts. Those are **draft requirements**,
not verified live API behavior. The [mock-v2 contract](README.md) defines concrete
routes and errors for implemented fixture kinds. Every live row is blocked on a
published platform schema plus milestone 10's installed/live evidence.

| Resource / capability | Parent scope and draft operations | Deletion / dependencies | Fixture/provider owner | Live prerequisite and platform owner |
| --- | --- | --- | --- | --- |
| Target and capabilities | GET scope/capabilities; explicit version/principal/kinds | No mutation | Milestone 2 | Published management target/version negotiation and machine auth; platform contracts and identity milestone 2 |
| Project | Organization projects: CRUD, exact/list reads, import | Protection; environments/grants/pending children restrict delete | Milestones 2–4 | Stable customer project/tenancy API; platform identity milestone 2 and data milestone 6 |
| Environment | Project environments: CRUD, exact/list reads, import | Protection; buckets/grants/pending children restrict delete | Milestones 2–4 | Explicit environment lifecycle and isolation; platform milestones 2/5/6 |
| Service account | Organization service_accounts: CRUD, exact/list reads, import | Protection; dependent grants restrict delete; issuance remains external | Milestone 4 | Automation principal lifecycle/credential policy; platform identity milestone 2 |
| Grant | Organization grants: CRUD, exact/list reads, import | Exact owned grant only; no implicit grant from binding | Milestone 4 | Product permission adapter, revocation/freshness, attribution; platform identity/data milestones 2/6 |
| Bucket | Environment buckets: CRUD, exact/list reads, import | Protection, nonempty and dependent grants; no force empty | Foundation; expanded service settings milestone 6 | Customer object service/data API, retention/quota; platform data milestone 6 |
| Artifact registry / digest reads | Proposed environment-scoped service lifecycle; routes unresolved | Retained/in-use artifacts and explicit retention | Milestone 5; no fixture/provider resource yet | Registry lifecycle, OCI push/pull and immutable digest policy; platform artifacts milestone 4 |
| Database / reads | Proposed environment-scoped service lifecycle; profiles/routes unresolved | Data protection, retention, consumers | Milestone 6; no fixture/provider resource yet | Managed database profile/readiness/data API; platform data milestone 6 |
| Secret reference | Proposed externally owned secret identifier/version; routes unresolved | Remove reference only; preserve secret/consumers | Milestone 7; no fixture/provider resource yet | Resolution authority/rotation contract; platform workload/data milestones 5/6 |
| Service binding | Proposed exact consumer/service connection; routes unresolved | Explicit detach; no implicit authorization/cascade | Milestone 7; no fixture/provider resource yet | Dependency graph and connection contract; platform workload/data milestones 5/6 |
| Container service | Proposed digest/configuration desired revision; routes unresolved | Retire compute; retain independently owned services | Milestone 8; no fixture/provider resource yet | Accepted deployment, active revision, health and recovery; platform workloads milestone 5 |
| Endpoint | Proposed exact workload routing/TLS lifecycle; routes unresolved | Remove route, preserve target/data; custom domains later | Milestone 8; no fixture/provider resource yet | Public/private ingress policy, TLS and reachability; platform workloads milestone 5 |
| Operation / management receipt | Scoped GET; owner/key/generation/fingerprint; tombstone on expiry | Pending cannot expire; transfer unsupported | Milestones 2–3 | Durable acceptance, fingerprint encoding, independent owner identity, retention/transfer and real effect recovery; platform contracts and recovery milestone 8 |

Upstream owners are the platform's
[identity](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/roadmap/02-identity-and-reactive-ingress.md),
[artifacts](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/roadmap/04-nix-artifacts-and-isolated-builds.md),
[workloads](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/roadmap/05-workload-isolation-and-runtime-bindings.md),
[data](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/roadmap/06-dataset-access-and-published-computation.md)
and [recovery](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/roadmap/08-durability-recovery-and-upgrades.md)
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
