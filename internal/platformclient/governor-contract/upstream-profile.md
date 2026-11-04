# CI governor management profile

**Provisional contract, not an implemented Platform endpoint.** The
[schema](ci-governor-v1.schema.json) and [independent vectors](ci-governor-v1.vectors.json)
define `spoke.platform.http.ci-governor.v1` / `ci-governor-v1` for client
conformance. They do not extend the bucket or publication profiles. No native
owner, worker, GitHub adapter or billable execution is qualified here. The
[service contract](../ci-demand-and-governor-v1.md) and
[delivery plan](../../docs/roadmap/ci-runner-implementation.md) retain those gates.

## Scope, authority and transport

Let `S = /v1/ci/organizations/{organization}/projects/{project}/environments/{environment}`.
All three scopes must already exist. The selected environment is an operator
infrastructure scope, not the caller's workflow repository. IDs are opaque
lowercase ASCII identifiers as bounded by the schema; they are never names to
resolve through another API. Bearer credentials remain external to Terraform.

Every request uses HTTPS with verified TLS and includes `Authorization: Bearer …`,
`X-Spoke-Organization`, `X-Spoke-API-Version: ci-governor-v1`, and
`Accept: application/json`. Bodies require `Content-Type: application/json`.
Every response, including errors, has one exact version header, one
`X-Spoke-Management-Profile: spoke.platform.http.ci-governor.v1`,
`Cache-Control: no-store` and JSON content type. Redirects, browser cookies,
Origin and forwarded authority are refused. Credentials never follow redirects.

Current authentication supplies principal and scope; bodies cannot select them.
Only operators can prepare/commit/abort policy. Authorized readers can observe
their principal's declarations and receipts, but customer/job credentials cannot
read the operator inventory or mutate it. Import is observation, never authority
transfer. Same credentials plus copied keys are indistinguishable from recovery.
Revocation denies new requests and settlement rechecks the accepted subject;
independently authorized cleanup must retain/refuse the original operation.
That native cleanup path is an unqualified Platform prerequisite.

## Routes and finite bounds

| Request | Successful result |
| --- | --- |
| `GET S/capabilities` | 200 `Capabilities`; explicit provisional status and unsupported execution/billing/warming/handoff |
| `GET S/governors/{id}` | 200 `Governor`, or authoritative `resource_not_found`/404 |
| `GET S/governors[?cursor=…]` | 200 `GovernorPage` at one snapshot |
| `GET S/management/{key}` | 200 latest `Operation` for this stable management key |
| `GET S/management?state=pending[&cursor=…]` | 200 `PendingPage` containing prepared or accepted requests only |
| `GET S/requests/{key}/{sequence}` | 200 exact retained `Operation` |
| `PUT S/requests/{key}/{sequence}` with `Intent` | 201 newly retained prepared `Operation`; 200 exact replay, regardless of its later status |
| `POST S/requests/{key}/{sequence}/commit` with `Commit` | 202 accepted, or 200 an already terminal `Operation` |
| `POST S/requests/{key}/{sequence}/abort` with `Commit` | 200 aborted prepared request; 409 if already accepted/applied/refused |

PUT/commit/abort responses carry `Location: S/requests/{key}/{sequence}`; it is
the original request path, never an external URL. All errors use the schema's
`Error` with exactly `code`; the vectors own each code's HTTP status mapping.
Only `request_not_found`/404 proves absence of a request; only
`resource_not_found`/404 proves absence of a governor. Foreign objects are
forbidden. An incomplete parent listing cannot establish child absence.

Limits per scope: 32 non-retired governors, 64 management keys and 256 retained
requests/tombstones. No eviction or key recycling to admit new work. Bodies are
at most 16 KiB, responses 128 KiB, JSON nesting at most 24. Reject unknown or
duplicate keys, trailing values, invalid nulls, non-integral/out-of-range numbers,
unsupported combinations and unknown queries. All policy fields are required;
there are no implicit defaults. Counters are unsigned 32-bit integers; exhaustion
is a typed capacity refusal, never wraparound.

Pages contain at most eight entries, sorted by stable ID. Cursors are opaque,
scoped to principal, route/filter and snapshot; they expire on a changed cut with
`stale_snapshot`. `complete=true` requires an empty cursor; an incomplete page
requires a nonempty, nonrepeated cursor and items. A client returns no partial
collection on failure. Reading governors visits at most four pages and pending
requests at most eight; at most one pending request exists per management key.
Server work uses bounded indexes, not an unbounded receipt-history scan. Every
item repeats scope/principal. GET never prepares, commits, settles or cleans up.

## Durable prepare and commit

The caller supplies a stable `management_key` for a logical governor and an
explicit `sequence` for each logical change. Sequence zero starts the key;
successors must be exactly one greater than its latest terminal request. A
prepared/accepted request prevents a successor. Never generate a new sequence
as a timeout recovery strategy. The latest request's full intent is discoverable
by management key even when Terraform saved no resource or operation ID.

1. PUT atomically binds the authenticated principal, scope, key, sequence and
   exact canonical intent to a retained operation ID, before responding. It also
   reserves sole policy ownership of the referenced admission entitlement for a
   new governor. It changes no governor policy and launches no worker. Another
   governor or generic scaling owner of that entitlement conflicts.
2. An exact repeated PUT returns the same ID, fingerprint, intent and current
   status. Changed content at the same sequence returns `intent_conflict`.
   Foreign ownership returns `forbidden`; a missing/skipped successor returns
   `sequence_conflict`. Admission-policy ownership is immutable while a governor
   exists. Unsupported handoff is explicitly refused.
3. Commit checks the retained fingerprint and current authorization, then durably
   records `accepted` before any policy effect. It never creates an absent
   request. Repeated commit returns its original operation. Settlement validates
   the original expected revision; stale intent receives a retained `refused`
   result, never an updated revision fence. Accepted is not applied or ready.
4. Abort is explicit and only valid while prepared. It durably records `aborted`
   and releases an unused create entitlement reservation. It cannot cancel
   accepted work. A client timeout/disconnect never implicitly aborts anything.

The first create uses a null target and revision zero. Updates and retirement
name the exact owned target and positive expected revision. Target identity is
immutable; later requests for a key cannot choose a different governor. Applied
create/update increments its desired revision once; observation and execution
counts remain controller-owned. A → B → A uses three distinct sequences and
revision fences, even though A's policy bytes recur. A refused/aborted operation
consumes its sequence; a later authorized change explicitly chooses a successor.

After successful retirement the management key is a permanent tombstone;
recreation needs a new key and new governor identity. Original requests remain
readable. Administrative receipt expiry applies only to terminal requests and
retains the identity/fingerprint tombstone: subsequent GET/PUT/commit/abort
returns `receipt_expired`/410 and never re-executes. Pending work cannot expire.

An adapter must recover the latest exact request before choosing a new logical
change, including before state save on create/update/delete. Persisting random
keys only in Terraform state, auto-incrementing on retry, or using a policy hash
as identity violates this contract. The Go client accepts explicit keys/sequences;
the later Terraform adapter must prove its own selection/recovery protocol.

## Canonical commitment

Fingerprint = SHA-256 of UTF-8 `spoke.platform.http.ci-governor.v1`, newline,
then compact JSON of `{scope, principal_id, management_key, sequence, intent}`
with every object key sorted lexicographically, ASCII strings, decimal integers,
and no whitespace. Arrays retain order; the initial execution-profile list is
exactly `["linux-x86_64-v1"]`. Full Intent includes explicit null target/policy
where the schema requires them. Input object order/whitespace do not matter.
Independent Python vectors include create, reduction, return to the original
policy and retirement. Client and server must validate typed input before hashing.
No native codec, signing secret or Transit dependency enters the HTTP client.

## Policy and retirement

Policy references already-authorized admission, budget and cache policies; callers
cannot create those objects through this API. References cannot expand account
authority or invent spendable credit. The prepaid-credit foundation is a separate
native/billing contract; its test reservations are not live CI admission.

Initial execution compatibility is Linux/x86_64 with zero warm runners. The
capability matrix explicitly advertises `execution=false`, `billing=false`,
`warm_capacity=false`, `handoff=false`. Unsupported profiles, nonzero warming and
successor transfer must fail rather than silently degrade. No successful
management result implies physical supply or qualified profiles.

Increasing a ceiling without demand creates nothing. Reductions affect future
admission; existing leases retain original limits, profile, deadline and usage
terms. A temporarily over-limit observation is valid and cannot be overwritten
by refresh. Readiness, active revision, counts and blockers are observations.
`active_revision` cannot exceed desired revision; readiness can remain disabled
after policy applies. Single ownership and aggregate account/operator quotas
across repositories must be enforced by the future admission owner.

Protected retirement produces a retained refusal. Otherwise retirement disables
new admission and enters draining. Uncertain leases, allocated capacity and
unsettled usage retain reservations and prevent successful retirement. Only
authoritative terminal evidence permits completion and governor absence. Caches,
artifacts, usage and receipts survive retirement. Independent cleanup after
caller revocation, physical shutdown and atomic quota enforcement need actual
retained-native and external-peer qualification; synthetic conformance cannot
satisfy them.

## Implementation and acceptance boundary

`tests/ci-governor-contract.test.py` validates this schema, response vectors,
canonical identity and negative boundary cases in ordinary checks. The provider
will copy these exact committed bytes, build a deterministic persisted HTTPS
peer, and exercise an unregistered Go client. No service route is installed by
this contract. TF17/P4/P5/P19 remain unfinished; Platform #56 owns the native
management, assignment, fleet, cache and commercial proofs that follow.
