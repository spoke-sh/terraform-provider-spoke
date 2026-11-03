# Independent publication consumer and declaration gate

The explicit `publication-declarations` gate passed all 42 required case groups
on 2026-11-03 at 06:32:39 UTC. The [machine record](publication-declarations.json)
retains the exact source, executable and consumer-lock identities. Fixture execution took
1,473.096 seconds. No skipped or missing test can satisfy this gate.

| Selected input | Executed identity |
| --- | --- |
| Installed provider | `bc1860930924ee5c0a88004273eb85bedf54cfe2` |
| Provider version | `0.1.0-dev.gbc1860930924ee5c0a88004273eb85bedf54cfe2` |
| Provider SHA-256 | `ad5897b486805411860785fc4bdf2689ab1f15e1d4bc1bf409c8f170d51646ea` |
| Clean qualification harness | `37c5265150a146663abb8bcbb698ed14bcc82bde` |
| Clean Platform fixture | `65f8284d0197ccc0ed450dc8c3e0b074947979cb` |
| Declaration profile | `spoke.platform.http.publication.v1`, `publication-v1`, development declarations |
| Schema SHA-256 | `db04fc4ce0e14534c2e44cef932b197e6d9aeb48006f21f7a1523c21def0511f` |
| Tools and runtime | Go 1.26.7, Terraform 1.15.3, Linux amd64 |

The artifact intentionally predates the harness. It is the exact committed
package already qualified by the [installed graph](publication-graph.md), now
selected by the independent consumer's flake, exact constraints and locks.
The harness compiles its test driver separately; Terraform never recompiles the
provider or uses a development override. Native bootstrap and service ownership
stay in Platform's normal fixture. The provider uses its HTTPS management API.

## Executed coverage

- A complete copy of `examples/installed-consumer` outside the provider checkout
  resolved the same Nix mirror bytes. Catalog, nightly and production initialized
  with committed readonly checksums and separate local state/credential files.
  Validate, apply, empty plans, target imports, isolated policy updates, protection
  and teardown passed. The record includes the Nix lock and four Terraform-lock
  hashes. Terraform's PATH contained no compiler.
- Installed application/binding/target lifecycle, imports, drift, stale saved
  plans, dependency holds, scoped denial and observational reads passed. The
  separate installed-alias case preserved environment configuration isolation.
  The graph case scanned state for fixture credential values.
- All 24 actual provider-process death schedules passed: application create,
  update and delete; binding create and delete; target create, update and delete,
  each before submission, after acceptance and after settlement before state save.
  Retained native services restarted separately before recovery. Original
  operation/resource identities survived.
- Six additional native groups passed: same-principal credential rotation and
  revocation; target tombstone with denied release and binding recovery; original
  binding-admission revision conflict; expired receipt refusal without new
  identity; application/receipt lifetime exhaustion with complete 64-receipt
  discovery; and concurrent same/conflicting intent without duplicate resources.

The harness verifies actual Go test events as well as process exit status. Its
separate [case inventory](../internal/testutil/liveplatform/declaration-cases.json)
rejects missing, duplicate, skipped or failed cases. Profile, source, freshness,
artifact and case validation reject mismatched reports. Release validation still
rejects this local evidence kind.

## Reproduction and other checks

Use the explicit configuration and command in the
[declaration-gate instructions](../docs/platform-http.md#publication-declaration-gate):

```sh
SPOKE_LIVE_CONFIG=/absolute/declarations.json \
SPOKE_LIVE_REPORT=/absolute/new/declarations-result.json \
  nix develop --command just test-acc live
```

The recorded run selected the identities above with a 1,800-second deadline and
fresh isolated fixtures. `TMPDIR` selected private workspace scratch storage;
the runner uses a short private socket path beneath the user's cache for nested
Nix shells. The aggregate live report passed and referenced this detailed report.

The locked shell's `just fmt`, `just check` and `just build` passed. The full
check includes Go vet/race unit tests, workflow/format/flake checks, Python report
and release rejection tests, and mock Terraform acceptance (1,231.279 seconds).
Root and consumer
`nix flake check --all-systems --no-build --no-update-lock-file` passed; this is
shell evaluation, not additional runtime qualification. The copied bucket consumer
also passed with the updated exact pin and readonly Random/Spoke checksums.
The exact artifact's full `just test-acc installed` regression remains attributed
to the earlier [graph run](publication-graph.md); this change adds qualification
harness and consumer behavior without changing that provider executable.
Documentation generation and relative-link/status/badge checks passed in the
final documentation follow-up.

Earlier attempts are not qualification evidence: an initial nested temporary
path exceeded the Unix socket limit, an interrupted run produced no final report,
and later preflight/setup attempts encountered a removed scratch checkout and a
stale Nix evaluation-cache derivation. The socket-path fix is in the recorded
harness; the exact fixture and consumer shell were recreated before this complete
run. No artifact/profile pin or service semantics changed to bypass a failure.

## Remaining boundaries

This qualifies local, operator-provisioned declarations only. Runtime-head policy
preservation and selected-head delete refusal have synthetic coverage in the
[graph record](publication-graph.md); the normal API cannot select a runtime head.
The independent roots use local state, not a hosted encrypted/locked state service.
Identical principal/key clones cannot be distinguished from recovery and require
external state locking. Lifetime capacity is not reclaimed by deletion or expiry;
single-apply replacement requires a new management key.

Object upload/sealing, immutable HTML/JS/Wasm delivery, preview/promotion/rollback,
custom domains/TLS, other runtime platforms and Registry release remain
unqualified. TF18 stays In Progress; TF19 stays Planned. Current upstream workflow
results and their independent blockers are recorded in the
[Platform adoption plan](../docs/roadmap/platform-adoption-implementation-plan.md).
