# Provisional lifecycle families (mock-v5)

This is a provider development contract, **not a published Platform API**.
Platform `7af27c2` exposes only native-v2 bucket metadata. The fixture's clock,
DNS, peer consent, synthetic data, demand and writer controls are test utilities;
they are neither HTTP service endpoints nor provider-side controllers. Management
reads never advance those controls. No data-plane claim follows from these tests.

All twelve kinds use the existing environment-scoped create/update/delete,
exact/complete-list, canonical import, current authorization, revision conditions,
principal-bound management keys, retained receipts and tombstones. They share the
same installed-provider recovery path. Names are mutable and are never adoption
keys. Protection defaults to true, independent dependencies restrict deletion,
and retained data cannot be erased by Terraform. The precise required JSON
members and union types are in [mock-v5.schema.json](mock-v5.schema.json).

| Family | Desired declaration | Independent observation and ownership |
| --- | --- | --- |
| Job / schedule | Pinned workload image/identity/configuration, 1–64 concurrent runs, 0–10 retries, 1–86400 second deadline; standard five-field cron, explicit IANA zone, enable, skip/catch-up-once, allow/forbid overlap | Scheduler owns immutable run ID, captured job revision, attempt and outcome. Execution output and secrets are excluded. |
| Stream / queue | Stream partition count 1–64 and partition ordering; FIFO queue visibility 1–86400 seconds, 1–100 attempts; bounded capacity and retention | Messaging owners retain payloads, offsets, leases and outcomes. Bindings target exact stream/queue IDs and require explicit produce/consume grants. |
| Volume / attachment | Filesystem/block, positive capacity, exact region, single-writer access; exact workload, volume and mount beneath `/data/` | Attachment controller owns writer epoch and completion. Mount requires `volume_mount`; revocation fences access. Detach retains data; stale writer epochs are refused. |
| Domain / private connection | Lowercase ASCII domain and endpoint; local workload, exact peer organization/network and canonical private IPv4 /16–/30 | External DNS owner supplies public challenge proof; TLS keys remain external. Peer consent is independent. No public fallback. |
| Backup policy / clone | Exact database, cron/zone, retention, enable; exact completed recovery point and source database, immutable clone capacity | Recovery service owns points and their exact source revision. Source updates cannot retarget a clone. Synthetic clone writes are isolated from source writes. |
| Scaling / placement | One policy per service: min/max 0–1000, CPU target, cooldown; exact region and stopped strategy | Controller owns observed instances/location. Removing a policy freezes the last observation; it never removes the workload or data. |

Capacity defaults to 1 GiB and is bounded at 1 PiB. Retention is 0–36500 days
(backup policies require at least one day). Growth is permitted for messaging and
volumes; shrinking is refused. Stream partition/order, volume profile/region/mode,
attachment selectors, domain/endpoint, connection selectors, schedule job,
backup source, clone declaration and policy target require replacement. Protection
and nonempty/dependency restrictions still apply to replacement.

## Scheduler and history

Schedules start disabled. The explicit fixture clock admits at most one due
occurrence per schedule per advance. `skip` ignores missed occurrences;
`catch_up_once` admits the oldest due occurrence and advances to the next future
one. Disabled, unauthorized, unready or concurrency-blocked occurrences are
skipped, never queued without bound. Overlap `forbid` applies across all schedules
of the selected job. Admission captures revision and immutable run identity before
persisting; restart cannot redispatch that occurrence as new work. Failed attempts
may retry within the configured limit under that same identity. Timeout or unknown
external effects remain `unknown`, occupy concurrency and block retirement until
the original execution is resolved. Editing a definition affects future admission;
editing a schedule resets its next occurrence without changing accepted runs.

Cron uses the pinned [robfig/cron](https://github.com/robfig/cron) five-field parser
with embedded IANA timezone data. Nonexistent DST wall times are skipped; repeated
wall times are distinct UTC occurrences. No descriptors, interval expressions,
seconds field or implicit `Local` zone are accepted.

`GET jobs/{id}/executions` and `GET backup_policies/{id}/recovery_points` return
complete, ordered metadata for one authorized parent and exact scope. The fixture
retains at most 2048 executions and 2048 points across its ledger. Budget exhaustion
fails the clock transition atomically; it never truncates a successful response.
The HTTP client also bounds response bytes, time and item count. Missing parents,
denial, incomplete/unbound responses, duplicate IDs and invalid outcomes fail.
Terraform exposes `spoke_job_executions` and `spoke_recovery_points` data sources;
individual executions and points are not managed resources. Histories contain
neither payloads nor backup contents. Point expiry is an explicit independent
fixture-owner action and respects retention plus existing/accepted clones.

## Network, attachment and policy transitions

Fixture domain challenges expire after one hour. Controller advancement rotates
an expired challenge and requires a new external proof; proof removal clears
verification. Duplicate domain claims, duplicate volume writers/mount slots,
overlapping connection routes and competing policies are rejected, including
accepted mutations. Private consent is bound to the exact scoped connection, not just its remote
network; consenting to one relationship cannot authorize another. Revocation clears readiness. TLS failures can
be injected separately; no real DNS, certificate issuance or traffic is exercised.

Detach settles under the fixture lock while advancing the writer epoch before
publishing completion. Writer replacement preserves synthetic data and refuses old
epochs across ledger restart. A real backend must supply durable host/storage fencing;
these single-process controls do not qualify distributed exclusion or durability.

Scaling advances by one instance per controller step, clipped to min/max, cooldown
and explicit quota. Missing metrics or observations older than 300 seconds preserve
capacity. Placement supports fixture regions `region-a` and `region-b`; incompatible
attached-volume placement is refused. The stopped transition is synthetic: drain,
physical relocation, routing continuity and distributed fencing still require the
upstream service and live evidence.

## Remaining upstream gates

Stream/queue payload processing, lease settlement, dead-letter behavior and mixed
transactions are not implemented by this management fixture. Native data-plane
integration belongs to Transit and Platform. Likewise, the fixture's synthetic
point is not a whole-database backup or a real restore, and its volume canary is
not a filesystem. These remain explicit gates in milestones 12–17. The selected
clone profile is same-environment only; cross-scope sharing, writable Table branch
semantics and live stateful moves are not advertised.

Hard cutover: only mock-v5 and Terraform resource schema version 3 are accepted.
Do not migrate old ledgers/state or reinterpret unresolved old work. Retain the
original immutable artifact to inspect old operations. The native-v2 codecs have
been removed; every family uses the same Platform HTTP client with no alternate
transport. Actual service qualification still requires a Platform-owned contract.
