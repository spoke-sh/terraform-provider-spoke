# Provisional management contract: mock-v1

This fixture contract belongs to the provider's experimental test suite. It is
**not a published Spoke Platform API**. Platform source
`78cebeb173916810c107d6a90ed5711f63f4d742` has no executable customer provisioning
contract. Track platform agreement and replacement of this fixture in
[implementation issue #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).

`mock-v1.schema.json` specifies the JSON shapes. Handwritten Go boundary types
are checked against it by fixture tests. Generate types from the platform-owned
schema when that contract exists. The server and client share boundary types,
but independently implement admission and recovery; raw HTTP tests check the
fixture without using client mutation helpers.

Every request uses HTTPS, `Authorization: Bearer <external-token>`,
`X-Spoke-Organization`, and `X-Spoke-API-Version: mock-v1`. Every response carries
the same API version header. The test server trusts only synthetic credentials
for its seeded organization/project/environment. This models a proposed machine
credential, not browser authentication or a qualified production auth scheme.

Let `S = /v1/projects/{project_id}/environments/{environment_id}`.

| Method and path | Successful response | Behavior |
| --- | --- | --- |
| `GET /v1/projects/{project_id}` | 200 Project | Exact scoped discovery |
| `GET S` | 200 Environment | Exact scoped discovery |
| `GET S/buckets/{bucket_id}` | 200 Bucket | Observational, authoritative resource read |
| `POST S/buckets` | 202 Operation | Accept create with a Mutation body |
| `PATCH S/buckets/{bucket_id}` | 202 Operation | Accept update with a Mutation body and expected revision |
| `DELETE S/buckets/{bucket_id}` | 202 Operation | Accept delete with a Mutation body and expected revision |
| `GET S/management/{management_key}` | 200 Management | Inspect the latest durable receipt and current resource binding |
| `GET S/operations/{operation_id}` | 200 Operation | Observe the original accepted operation |

This first contract supports exact-ID discovery only. Lists/pagination,
capability negotiation beyond the version header, and other service APIs are
future work. Unexpected requests fail fixture tests.

## Identity, retries, and recovery

IDs are opaque and scoped. Bucket names are mutable labels. A management key is
a non-secret caller-chosen ownership identity, stable across provider crashes.
Use a unique key per independently managed resource; copying both the key and
credentials into another state gives it the same authority. The provisional
protocol cannot distinguish that copy from recovery by the original owner.
Never silently use a name as a management key.

An intent contains `kind`, `management_key`, `target_id`, `expected_revision`,
and (for create/update) `spec`. The request fingerprint is lowercase SHA-256 of
UTF-8 compact JSON in that field order. `spec` orders `name` before
`deletion_protection`; delete omits `spec`. The golden serialization test records
this provisional convention; it is not a reimplementation of Transit signing.

Before dispatch, inspect the management receipt. Resume a matching operation,
or reject different intent while an operation is pending. A new mutation must
compare-and-swap the last `expected_generation`; accepting it increments that
generation and stores its fingerprint/operation before any completion. Replay
of that accepted generation with an identical fingerprint returns its receipt.
Conflicting generations or revisions return 409, and do not change intent.

Create has empty `target_id` and revision zero; update/delete require the exact
resource ID and revision. Revisions fence changes, including a later return to
old configuration. The management binding survives deletion. After deletion, a
new create with the same management key uses a new generation and object ID.
Receipts are retained for the fixture's lifetime; production retention and
ownership handoff require a platform contract.

Acceptance returns an operation, not completion. The fixture controller may
complete immediately or be held by the test. GET never advances operations.
Pending creates are not authoritative absence: their object read returns 409
`operation_pending`. An operation timeout preserves uncertainty and its receipt.
Tests can drop a reply after acceptance, advance operations, revoke access,
change resource state, or remove an object through Go controls, not HTTP API
endpoints. The in-memory ledger outlives client instances but not the fixture
process. Durable server restart qualification remains future work.

## Errors and protection

| Status | Codes / meaning |
| --- | --- |
| 400 | `invalid_request`, `incompatible_api` |
| 401 / 403 | `unauthenticated` / `forbidden`; no authoritative absence |
| 404 | `resource_not_found` or `management_not_found` only after scope authorization; unknown routes are `unsupported_route` |
| 409 | `generation_conflict`, `revision_conflict`, `ownership_conflict`, `operation_pending`, `deletion_protected`, `not_empty` |
| 429 / 503 | Reserved client transient handling: `rate_limited` / `unavailable` |

The client bounds body sizes and retries, verifies response scopes/identities,
and does not print arbitrary remote error messages. Deletion requires explicit
disabling of deletion protection and an empty bucket. The fake tracks only an
empty/nonempty flag; it does not implement S3, object uploads, or retained data.
No cascade or force-empty operation exists. Bindings/dependency graphs, quota
accounting, live service guarantees, and production receipt durability require
separate contract and implementation work.
