# 6. Databases and object storage

[Roadmap](README.md) · **Status:** Completed

Let customers manage database and bucket services with explicit capacity,
retention, access and data protection independently of application compute.

See the [implementation plan](milestones-5-8-implementation-plan.md#milestone-6-databases-and-expanded-buckets)
for contract work, state compatibility, delivery steps and test gates.

## Delivered evidence

Implemented against provisional mock-v3 in revision
[`201bf93`](https://github.com/spoke-sh/terraform-provider-spoke/commit/201bf93a3496cca475a3c5746d3114305a8bc645).
The [milestones 5–8 qualification record](../../qualification/milestones-5-8.md)
links executed Terraform, HTTP, installed-process recovery and complete-graph
checks. Generated reference and examples describe the implemented experimental
surface. Completion is provider/mock qualification; real service behavior remains
an explicit [milestone 10](10-live-platform-qualification.md) gate.

## Prerequisites and scope

Use [scope and access](04-scope-and-access.md) and shared contract/recovery gates.
The prior bucket baseline had only name, protection and identity/revision lifecycle;
the delivered extension adds capacity, retention and safe endpoint observations.
Platform owns storage provisioning and the supported database/object interfaces.
Registry work is not a prerequisite for this independently reviewable slice.

## Delivery sequence

1. Define database profiles, tenancy, capacity, retention, endpoint discovery,
   permitted changes and replacement semantics. A Transit SQL wire interface does
   not establish PostgreSQL or MySQL engine compatibility.
2. Implement `spoke_database` and exact scoped reads after its schema and mock
   operations. Distinguish accepted provisioning from usable service readiness;
   return connection metadata without embedding credentials.
3. Expand `spoke_bucket` only with contract-backed settings: quota/capacity,
   retention, access and data-plane endpoint identity. Define immutable choices
   and state migration for any changed schema. Do not expose backend cloud knobs.
4. Model stored data and dependencies as test observations sufficient to reject
   destructive changes, without pretending the fixture is an object/database
   data plane. Preserve protection through update, import and interrupted delete.
5. Record the optional `spoke_bucket_object` decision. If later selected, give it
   a separate milestone for bounded deployment files, exact object/version and
   digest, external content transport and state without file bytes. It is not
   required for this milestone; ordinary datasets and files use the data API.

## Acceptance criteria

- Database and expanded bucket lifecycles pass import, empty-plan, drift, denied
  access, stale revisions and interrupted CRUD through Terraform and HTTP.
- Resource IDs survive allowed changes. Replacement, capacity reduction and
  retention changes cannot silently discard existing data or retarget another
  owner's service. Protected/nonempty/in-use deletion fails explicitly.
- Service retirement preserves separately owned data services. Connection
  outputs and failure diagnostics contain no credentials or secret values.
- Examples distinguish database service management from schema migrations/rows,
  and buckets from POSIX volumes or internal Transit segment storage.
- Actual bucket upload/download and database access remain required in
  [milestone 10](10-live-platform-qualification.md), with provider schema fields
  promoted to supported only after their backend contract and evidence exist.
