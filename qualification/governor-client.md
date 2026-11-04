# CI governor synthetic HTTP client conformance

The unregistered Go client passes all 12 required groups against an independent,
persisted synthetic HTTPS peer. This is **synthetic client conformance**, not a
Platform service, native runtime, installed Terraform or runner qualification.
The [implementation plan](../docs/roadmap/ci-governor-implementation-plan.md)
retains the later adapter and service gates.

## Exact evidence

The [machine-readable report](governor-client.json) records the command, case
results, elapsed times, source identities and raw Go JSON output digest. The run
completed at 2026-11-04 02:41:42 UTC. Implementation, tests, contracts and
build inputs match the commit below; only evidence/documentation was dirty. The
initial clean-source run at `2b808c5` is preserved separately in the report.

| Input | Identity |
| --- | --- |
| Provider client and peer source | `246ce7320a086dc7f4e46b069278c13c9fc2539f` |
| Platform contract source | `af60b8a13d7b9f953010552096e33174c22d3981` |
| Schema SHA-256 | `7899392a4553f748606a59bbccdce03b61edcb93ca6b101d6bf1a192114b15cd` |
| Vectors SHA-256 | `4c5a7b1732d245ae935bc4e996aee1f000a9832eddd3c629c16d490349b4c260` |
| Profile text SHA-256 | `f2b4561ed1e462320596172b9663cb56f7b1d162f8d2157c2244e713cba53cf1` |
| Profile / status | `spoke.platform.http.ci-governor.v1` / provisional |
| System | Linux amd64, locked Nix shell, race detector enabled |

The copied schema, vectors and profile are byte-identical to their committed
Platform source. Platform's branch subsequently merged main `33df0ac`, retaining
that contract commit and its bytes. The dependent provider PR must follow the
[owner contract PR #63](https://github.com/spoke-sh/platform/pull/63).
No main revision or runtime is adopted by this pin.

The client package completed in 24.377 seconds and the independent peer package
in 2.014 seconds; these are package durations, not service performance claims.
The [required-case inventory](../testdata/contracts/governor-client-cases.json)
is enforced by the test. Missing or extra groups fail it.

## Exercised boundaries

| Required groups | Evidence |
| --- | --- |
| Lifecycle/distinct intent; lost replies | Create, update, A → B → A and retirement retain one key/sequence per logical intent; lost replies before/after prepare and commit converge to the original operation. |
| Client death; deadline/resume | Twelve actual child processes are killed: create/update/retire each before dispatch, after preparation, after acceptance and after completion. The fixture separately reloads its ledger. Accepted deadlines preserve identity; explicit resume settles it without implicit abort. |
| Observational discovery; collection integrity | Exact/request/latest and multi-page governor/pending reads leave retained state unchanged; changed snapshots, duplicates, incomplete pages, loops and bounds return no partial result. |
| Authority/revision; retirement/tombstones | Customer/foreign callers, unauthorized references, competing owners and stale revisions are refused. Current revocation is rechecked at settlement. Protection, leases and unsettled usage block retirement; retained keys prevent reuse and preserve cache-policy references. |
| Abort/expiry; receipt identity | Explicit prepared abort survives a lost reply. Expired receipts remain tombstoned; mismatched commitments, changed identities and missing known receipts never authorize replacement work. |
| Malformed peers; TLS/secrets | Strict schema and canonical vectors, duplicate/null/unknown fields, unsupported capabilities, bad response headers, redirects, untrusted TLS, bounded failures and sanitized secret canaries. |

Five separate peer tests exercise raw wire requests, canonical commitments,
ownership/sequence, concurrent entitlement claims, revision/drain/tombstones and
incompatible-ledger refusal. The peer never imports the client. It persists state
before acknowledging acceptance or exposing a crash barrier; test controls seed
usage and settle it outside the public API. These controls implement no scheduler,
native coordinator, physical cache, GitHub assignment or cloud cleanup.

`TestGovernorClientHelper` is skipped as a standalone test because it is the
private subprocess entrypoint. The 12 parent recovery schedules invoke and kill it;
none of the 12 required conformance groups is skipped. Killing this client is
distinct from killing a Terraform provider process or native service node.

## Validation and limits

Platform's five independent schema/vector tests and
`nix develop --command just check` passed at contract `af60b8a` and again after the main merge at `f546ce9`.
This includes ordinary contract/build/Terraform checks, with no live deployment.

Provider `just fmt`, `just build`, focused race conformance and all-system Nix
evaluation without builds or lock updates passed. Full `just check` passed:
format/workflow checks, Nix packaging/install check, vet, race unit tests and all
44 top-level mock Terraform groups (1,234.310 seconds of acceptance). It started
at clean implementation `2b808c5`; the subsequent test-only expansion at
`246ce73` separately passed formatting, vet and the complete governor race suite.
The final build also passed; production code and build inputs did not change
during that test-only follow-up. Initial check attempts exposed
an omitted Nix source allowlist entry and unkeyed fixture literals; both were
fixed before the final run. Nix now includes all four embedded contract files.
No dependency, toolchain, consumer lock, workflow or existing profile pin changes.

The 26-resource/53-data-source Terraform surface is unchanged. The governor
client has no registered adapter or live qualification lane. Four-system shell
evaluation establishes no additional package/runtime support; distribution
remains Linux amd64. Existing installed/live evidence retains its original tuple.

Next: design and prove the Terraform adapter's durable key/sequence selection
across death before state save, planning, import, drift, replacement and drain.
Platform must separately establish retained native management and cleanup
authority, atomic account-wide admission, real GitHub assignment, physical
termination, cache isolation and usage/billing. The explicit execution, warming,
handoff and billing capabilities remain false. No release gate is completed.
