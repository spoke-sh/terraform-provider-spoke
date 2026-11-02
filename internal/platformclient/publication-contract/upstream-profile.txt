# Publication machine HTTP profile v1

Platform owns this JSON interface and all native integration. The
[schema](publication-v1.schema.json) and [independent vectors](publication-v1.vectors.json)
are separate from `mock-v5`/`spoke.platform.http.bucket.v1`; bucket clients and
provider schema are unchanged. The [implementation plan](../../docs/roadmap/publication-machine-api-implementation.md)
records scope, dependencies and the later C–I handoffs. Qualification is required
before advertising an installed endpoint.

## Negotiation, scope and supported routes

Send `Authorization: Bearer <machine token>`, `X-Spoke-Organization` and
`X-Spoke-API-Version: publication-v1`. The normal profile is
`spoke.platform.http.publication.v1`; the separate test executable uses
`spoke.platform.http.publication.peer.v1`. Responses identify their exact profile
in `X-Spoke-Management-Profile` and version in `X-Spoke-API-Version`, with
`Cache-Control: no-store`, JSON and `nosniff`. Consumers must reject an unexpected
profile; a peer response is never customer delivery evidence.

Let P be `/v1/publication/organizations/{organization}/projects/{project}` and
E be `P/environments/{environment}`. Scope is operator-provisioned and immutable
in native configuration. Organization/project identifiers have 1–128 ASCII
identifier bytes; this slice narrows environment identifiers and labels to 1–63.
Identifiers begin with an ASCII letter/digit, followed by letters/digits/`_`/`-`.
No percent-encoded paths, alternate spelling of numbers or browser credentials
are accepted. Resource IDs and counters are canonical decimal strings bounded
by the adopted native maximum `9007199254740991`. An ID never grants access.

| Method and route | Request schema | Result |
| --- | --- | --- |
| GET P/capabilities or E/capabilities | — | Capabilities; false flags must be honored |
| POST P/applications | CreateApplication | Original operation |
| GET P/applications or P/applications/{id} | — | Collection or Application; GET is the import identity |
| PATCH / DELETE P/applications/{id} | UpdatePolicy / DeleteDeclaration | Original operation |
| POST P/applications/{id}/target-bindings | CreateBinding | Reservation for the named environment and application revision |
| GET P/applications/{id}/target-bindings or P/target-bindings/{id} | — | Collection or Binding |
| DELETE P/target-bindings/{id} | DeleteDeclaration | Retained cancellation/release, after native child fencing |
| POST E/targets | CreateTarget | Consumes one exact project binding |
| GET E/targets or E/targets/{id} | — | Collection or Target |
| PATCH / DELETE E/targets/{id} | UpdatePolicy / DeleteDeclaration | Policy only; no selected-publication reset |
| GET P/operations/{id} or E/operations/{id} | — | Original Operation |
| GET P/operation-keys/{key} or E/operation-keys/{key} | — | Original Operation after a lost acceptance reply |
| GET P/operations or E/operations | — | Authorized Collection; optional state=all/pending/terminal |
| POST P/operations/{id}/expire or E/operations/{id}/expire | ExpireReceipt | Explicit operator command; eligible terminal receipts only |
| POST E/targets/{id}/promotions or /rollbacks | Selection | Peer profile only; fixed verified fixture commitments |
| POST E/operations/{id}/observations | Observation | Peer profile only; a bounded child GET under the original selection |

Mutations require `Content-Type: application/json`; DELETE has a JSON body.
Unknown/duplicate fields and headers, unsupported methods and ambiguous paths
fail closed. Machines call loopback HTTP through the operator's byte-preserving
TLS ingress. Strict Host checking and refusal of Origin, Cookie, forwarding and
principal-spoofing headers preserve the existing boundary.

## Authority and retained lifecycle

Native resource grants and retained per-owner product roles jointly authorize
access. Roles are reader=1, policy administrator=2, publisher=4, recovery=8 and
operator=16. These bits appear only in operator/native configuration, never as
client authority. All roles can read their granted scope; receipt discovery is
restricted to the submitting principal unless scoped recovery/operator authority
permits inspection. A recovery role cannot select a new publication. Current
native SEND and resource-read authority are checked during admission and again
before later policy/effect staging. Revocation does not cancel a committed call.

The project administrator reserves a binding in the catalog; an environment
administrator separately accepts target creation. Each owner uses its own original
accepted caller proof. Native authenticated calls consume/release the reservation;
no proof is forwarded to the wrong PID. One reservation cannot attach two targets.
Application deletion refuses reservations/live bindings. Target deletion retains
a tombstone before release. A denied/lost release leaves a conservative parent
hold; a fresh authorized binding deletion can inspect and release a tombstoned
child. Cancellation fences late consumption. Failed target creation can leave a
retained tombstone and binding hold visible to administrative recovery.

Protected declarations, pending targets and unresolved effects cannot be deleted.
This slice has no withdrawal capability: selected target references also block
deletion. All IDs, reservation slots and receipt namespaces remain consumed after
deletion/expiry. Bounds are finite installation capacity, not recyclable counters
or a claim of production scale.

## Original identity, failures and observations

An operation key has 1–32 identifier bytes and identifies one logical mutation in
one exact scope. First admission atomically retains its full normalized intent,
principal, original native input/ID and continuation. Expected revision/generation
failures after admission become terminal receipts. A new admission returns 202
and Location; a matching replay returns 200 and that same receipt. Reuse with
changed intent returns 409; another principal cannot claim the key. Token rotation
preserves the native principal and therefore ownership.

A lost submit/acceptance response returns 503 `outcome_unknown`, the original key
and its lookup URL. Lookup or retry the same request; do not allocate another key.
GET never schedules work. A terminal expired receipt returns 410 and retains its
identity/fingerprint and summary. Pending, unknown and live-referenced receipts
cannot expire. Receipt expiry is separate from artifact retention.

| State | Explicit observation result | Parent result |
| --- | --- | --- |
| Unknown selection | Exact complete operation/generation/bundle/configuration and changed ETag | Applied, same original identity; release its target fence |
| Unknown selection | Negative, partial, absent, unrelated or unchanged-ETag evidence | Remain unknown and fenced; another bounded observation may follow |
| Terminal conditional refusal (412) | Complete known bundle/configuration at a changed ETag | Refresh target precondition for a later decision; original refusal stays terminal |
| Applied selection | Any attempted new observation | Conflict; no rewrite of the outcome |

Each observation has its own operation key and expected observation revision.
One child is in flight per parent; up to four children use distinct finite GET
workers. Replaying a child does not send another GET. No child reissues PUT or
changes its original base ETag. A late capture cannot select a newer head. Peer
selection acknowledges provider state only: readiness stays pending, endpoint
null, and delivery capabilities false.

Stable pre-admission errors: 401 unauthenticated; 403 forbidden; 404 authorized
resource_not_found/unsupported_route; 409 operation_conflict/stale_snapshot;
410 receipt_expired; 413 body_too_large; 422 invalid_request/unsupported_capability;
429 rate_limited/capacity_exhausted; 503 outcome_unknown/incomplete_result/unavailable.
Malformed protocol/version uses 400 incompatible_api. A retained terminal refusal
is returned in a successful operation read, never disguised as a missing object.

## Bounds and complete reads

Eight applications, two environment owners, sixteen binding identities/project;
eight target identities/environment; 64 receipt identities/owner including
tombstones. At most 32 caller mappings and one concurrent submission per principal
in the exclusive gateway. HTTP control bodies are at most 64 KiB. The peer profile
has four selection slots with four observation slots each; unsupported data-plane
limits are never advertised as usable transfer capacity.

Collections return at most eight items, one checked native revision, `next_cursor`
and `complete`. A cursor is signed with the server-held native caller key over
scope/resource, principal, collection/filter, revision and position. It is not an
authorization token. Each page rechecks current resource access. Filtering precedes
counts and continuation. A changed snapshot returns 409 and requires restarting
the collection; never concatenate different snapshots into authoritative absence.
The final page alone has `complete=true`; it establishes completion only when
joined to earlier pages from that same snapshot. Terminal tombstones remain
visible in history as `{id,status:"expired",code:"receipt_expired"}`.

## Native codec and fingerprints

The fixed little-endian PMA1 command is 177 bytes; its STN1 cast frame is 190 bytes.
Field order: four magic bytes, kind u8 (the twelve kinds in schema order), key[32],
target/revision/related/generation/reserved-environment u64, name[63], bundle[32],
protection u8, retention u32. Strings are ASCII with zero padding; noncanonical
unused fields are rejected. Binding creation uses `related` for application ID
and `name` for the exact environment identifier; native process IDs never enter
that public intent. The reserved environment field is zero. Observation uses
`target` for parent operation and `generation` for expected observation revision.

Fingerprint = SHA-256 of the exact negotiated profile plus a zero byte,
`spoke.platform.publication.intent.v1` plus a zero byte,
SHA-256 of the exact UTF-8 scope path, then the complete PMA1 command. Full normalized
intent and key remain retained, so hashes do not replace ownership/content checks.
The vectors independently encode all twelve commands and their framed bytes.
This codec/storage version is separate from proof-v1 and bucket state; no implicit
conversion or deletion of an older installation is allowed.

## Follow-up routes with unavailable capabilities

The following schema envelopes define the C–F handoff; all routes currently refuse
with unsupported_capability. Their metadata does not enable byte transfer or
provider resources. Files and a potentially 512-KiB manifest travel through P6,
not the 64-KiB control endpoint. Future native payload staging needs its own gate.

| Route | Envelope and later gate |
| --- | --- |
| POST P/applications/{id}/uploads | FutureUpload: descriptor digest/count/byte reservation; C |
| GET P/uploads/{id} | Original session/revision, progress, expiry and immutable versions; C |
| POST P/uploads/{id}/renew | FutureRenewUpload; retains session identity and rechecks current rights; C |
| POST P/uploads/{id}/seal | FutureSeal: exact immutable manifest reference and digest; independent byte verification; C |
| GET P/bundles/{id} | Verified immutable versions/digests and explicit eligibility; C |
| POST/GET P/applications/{id}/publications; GET P/publications/{id} | FuturePublication; exact bundle/configuration identity and explicit public audience; D/E |
| POST P/publications/{id}/preview-authorizations | FuturePreviewAuthorization; audience/expiry bound to original publication; D |
| GET/POST E/domains; GET/PATCH/DELETE E/domains/{id} | FutureDomain/FutureDomainUpdate/DeleteDeclaration; F ownership, TLS and import gate |

Future read schemas and provider resources are finalized with their actual data
plane; consumers must not infer their shape from current declaration responses.
No protected-preview support, native route binding, hosted credential exchange,
GitHub Actions deployment or AWS adapter is delivered by this profile.
