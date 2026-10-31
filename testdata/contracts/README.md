# Provisional management contract: mock-v2

This provider-owned test contract is **not a published Spoke Platform API**.
The source review at platform revision
[`3f155dd`](https://github.com/spoke-sh/platform/blob/3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740/docs/platform-contracts.md)
still finds no public organization/customer provisioning lifecycle. The
[capability matrix](capabilities.md) assigns live prerequisites to platform owners.
[Milestone 10](../../roadmap/10-live-platform-qualification.md) gates live adoption.

`mock-v2.schema.json` specifies wire shapes; raw HTTP tests validate the fixture
independently of client mutation/retry helpers. Handwritten boundary types are
provisional until a platform-owned schema can be pinned and used for generation.
The [mock-v1 specification](mock-v1.md) and schema remain historical evidence.
The v2 header and routes deliberately reject v1; this is not a silent protocol
upgrade or a claim of compatibility with a published provider release.

## Target and authentication

Requests use HTTPS, externally supplied `Authorization: Bearer ...`,
`X-Spoke-Organization`, and `X-Spoke-API-Version: mock-v2`. Every response has the
same version header. A token maps to a stable principal and organization; a new
credential for the same principal retains its management ownership. Test-only
issuance controls never appear as HTTP endpoints or Terraform outputs.

Let `S` be one explicit parent scope:

| Scope | Path | Managed collection kinds |
| --- | --- | --- |
| Organization | `/v2/organizations/{organization_id}` | `projects`, `service_accounts`, `grants` |
| Project | `{organization_scope}/projects/{project_id}` | `environments` |
| Environment | `{project_scope}/environments/{environment_id}` | `buckets` |

IDs are opaque and names are mutable labels. A provider configured at a parent
may narrow its target for an individual resource; it cannot escape configured
project/environment assertions. Configuration performs no network mutation or
capability lookup. Every mutation first reads authenticated `GET S/capabilities`
and validates target, version, principal, supported kind and receipt policy.
The server independently checks current permissions at admission and completion.

## Operations and observation

| Method and route | Result |
| --- | --- |
| `GET S/capabilities` | Target scope, version, principal, resource kinds and `retain-or-tombstone` receipt policy |
| `GET S/{collection}` | Complete bounded paginated discovery, as specified below |
| `GET S/{collection}/{id}` | Exact authorized object with scope, kind, owner, key and revision |
| `POST S/{collection}` | 202 accepted create operation |
| `PATCH S/{collection}/{id}` | 202 accepted update operation with revision fence |
| `DELETE S/{collection}/{id}` | 202 accepted protected/dependency-checked deletion |
| `GET S/management/{resource_kind}/{management_key}` | Original owner-bound receipt and current resource binding |
| `GET S/operations/{operation_id}` | Original owner-bound operation status |

Mutations carry kind, management key, target ID, expected revision, spec (except
on delete), resource kind and expected management generation. Omitted resource
kind means bucket for the retained bucket intent serialization. The fingerprint
is lowercase SHA-256 of compact UTF-8 JSON in this order: `kind`,
`management_key`, `target_id`, `expected_revision`, optional `spec`, optional
`resource_kind`. Spec order is `name`, `deletion_protection`, optional
`subject_id`, `target`, `permission`. The golden test fixes this draft encoding;
it does not implement Transit signing. Authentication determines the owner;
clients cannot supply another principal in the mutation body.

Create uses empty target and revision zero; update/delete use exact ID and
positive revision. The server CASes generation/revision, persists intent and
receipt, then acknowledges acceptance. An identical prior-generation replay
returns the same receipt. Conflicting intent/generation is rejected. A matching
receipt is inspected before dispatch; uncertainty never causes a fresh write.
A later return to an earlier configuration uses a new revision/generation.

Accepted work is not completed work. Only explicit fixture controller actions
advance operations; GET does not. Reads of an object with pending mutation return
`operation_pending`, preserving known state. Terminal failures preserve the
receipt and do not change the object. Timeouts stop waiting and retain recovery
context; they do not roll back accepted work. Preconditions, permissions, quotas
and dependencies are checked again at completion.

## Ownership, import and receipt lifetime

The ownership namespace is `(scope, resource_kind, management_key)` and is bound
to the authenticated principal. Another principal cannot read or replay its
receipt or mutate its object, even with administrative service permissions.
Same-name objects with different keys are distinct identities. Read permission
alone does not authorize management or import into a managed resource.

Import accepts an exact scope-qualified object owned by the current principal;
it recovers that object's existing management key. Import is not ownership
transfer or adoption of an unmanaged object. Transfer is unsupported in v2 and
must not be emulated by changing a key/name or deleting state. The seeded parent
scopes are discovery fixtures, not examples of adopting unmanaged scopes.

Receipts remain until explicitly expired by test control. Pending receipts cannot
expire. Expiration leaves a durable owner-bound tombstone and returns 410
`receipt_expired`; it never becomes `management_not_found` and never permits key
reuse. The client requires operator/platform reconciliation instead of guessing
whether the old operation happened. Deleted-object bindings retain generations;
a later authorized create with an unexpired key gets a new object ID/generation.

**Limit:** two states with the same principal credentials and management key are
indistinguishable from recovery. Use distinct externally managed principals for
independent owners. The draft does not claim to isolate credential clones or
provide a transfer/receipt-retention policy approved by the platform owner.

## Bounded discovery

Lists return scope, resource kind, non-null items, positive snapshot revision,
`next_token` and `complete`. `complete` is true exactly when there is no next
page. The fixture orders opaque IDs, uses two-item pages and opaque server-side
cursors bound to principal, scope, kind and snapshot. A mutation invalidates old
snapshots; restart or cursor-cache eviction also returns `incomplete_result`.
A cursor never grants permission; every page rechecks current authority.

The client returns the entire bounded result or an error, never partial success.
It rejects repeated/oversized tokens, duplicate/wrong-scope IDs, changed snapshots,
null items and incomplete terminal pages. Bounds are 100 pages, 10,000 items,
1 MiB per response, three attempts per eligible read, and a 10-second request
limit, inside the provider's total operation timeout. Tests can lower bounds and
inject the wait policy. Retry-After is capped at ten seconds. Denied pages,
masked 404s, outages and truncated discovery cannot establish resource absence.

## Scope, identity and grants

Project/environment/service-account specs use name and deletion protection.
Grant specs additionally require exact service-account `subject_id`, a `target`
path in the same organization and `permission` (`read` or `manage`). Subject and
target are immutable; name/permission/protection can change with revision checks.
Grant targets are the organization or an existing non-grant object. Authority
extends to descendants of that exact target; `manage` includes read. Grant
administration requires organization management authority. Removing one grant
leaves all independent grants intact. Deleting a service account invalidates its
external credentials; credentials themselves are never managed resources.

Parents with children, service accounts referenced by grants, and resources with
grants targeting them refuse deletion. Pending child creates also restrict parent
deletion. Protection is checked independently; nonempty buckets cannot be deleted.
No purge, force-empty, cascade, secret issuance, binding or data-plane API exists.
Self-revocation can complete on the server while making its receipt unreadable;
Terraform must preserve uncertainty/state until inspection authority is restored.

## Errors and durable fixture state

| Status | Codes |
| --- | --- |
| 400 | `invalid_request`, `incompatible_api` |
| 401 / 403 | `unauthenticated`, `forbidden` |
| 404 | Authorized `resource_not_found` / `management_not_found`; unknown route is `unsupported_route` |
| 409 | `revision_conflict`, `generation_conflict`, `ownership_conflict`, `operation_pending`, `dependency_conflict`, `deletion_protected`, `not_empty`, `incomplete_result` |
| 410 | `receipt_expired` |
| 429 / 503 | `quota_exceeded`, `rate_limited`, `unavailable` (only eligible transient reads retry) |

Remote messages are never surfaced in diagnostics. Only an allowlisted absence
code with HTTP 404 is authoritative. Credentials are never forwarded through
redirects. Mutation request bodies are capped at 64 KiB.

Each fixture writes only synthetic state to a private test directory. Atomic
replacement, file/directory fsync and persisted generation/receipt precede
acceptance notification. Restart discards the heap and reloads that ledger while
retaining endpoint/TLS identity. This tests the draft recovery protocol, not
production durability, cloud effects or a real platform controller.
