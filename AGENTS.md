# Developing the Spoke Terraform provider

## Scope and source of truth

- Read `README.md`, the [roadmap](docs/roadmap/README.md), the affected milestone and
  `HANDOFF.md` before implementation. The roadmap owns current scope, sequence,
  acceptance criteria and status. The
  [implementation tracker](https://github.com/spoke-sh/terraform-provider-spoke/issues/1)
  preserves the original mock-endpoint plan and Transit issue #290's history;
  the handoff records historical source context.
- This repository owns the Go provider, platform API client integration, tests,
  examples, generated documentation, Nix packaging, and independent releases.
  Platform service APIs and provisioning belong in `spoke-sh/platform`; generic
  runtime primitives belong in `spoke-sh/transit`.
- Keep numbered milestone planning in `docs/roadmap/`, with consistent status and
  dependency links in its index. Use GitHub issues for implementation tracking
  and link executed evidence when closing milestones. Use the documentation
  generation script below to preserve hand-maintained plans. Check the destination
  repository's instructions before making cross-repository changes.
- The current provider implements scope/access plus artifact registries,
  databases, buckets, secret references, bindings, containers and endpoints, plus jobs/schedules, messaging, volumes, networking, recovery and policy declarations, with
  exact/collection, artifact and bounded history discovery against the provisional mock-v5 API.
  Read its capability matrix and qualification record before extending behavior.
  Do not report mock tests or skipped checks as qualified live services.

## Environment and everyday commands

Use the locked Nix shell: `nix develop` or `nix develop .#terraform`. For a single
command, use `nix develop --command just check`.

Keep the justfile at **no more than five public project tasks**, plus the private
default recipe that lists them. Extend the existing tasks rather than adding
another public command:

| Command | Use |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform, and Go files. |
| `just check` | Check formatting, workflows, and the flake; run Go vet, unit tests, and mock Terraform acceptance. |
| `just build` | Build the root provider executable into `bin/`. |
| `just test` | Run unit tests with the race detector and `TF_ACC` unset. |
| `just test-acc [mock/installed/live]` | Mock lifecycle; exact Nix artifact lifecycle/recovery; or explicit live qualification preflight. Live currently retains the superseded native-only blocker checks; replace them per the HTTP plan. |

Bare `just` lists the tasks. Missing source, Terraform, or mock acceptance cases
must fail checks. Never select live tests from ambient credentials or silently
fall back from live tests to mocks.

Keep the nixpkgs revision and `flake.lock` pinned. Update them deliberately and
verify shell entry after toolchain changes. Go libraries belong in `go.mod` and
`go.sum`; keep `GOTOOLCHAIN=local` so builds use the Nix toolchain. Nix evaluation,
builds, and shell entry must have no deployment effects or embedded credentials.

## Provider implementation

- Use Go Terraform Plugin Framework and protocol 6. Root `main.go` serves the
  provider; `internal/provider/` owns Terraform schemas/lifecycle and
  `internal/platformclient/` owns the HTTP management client. Platform owns
  Transit integration, native signing/proofs, internal sequencing and durable
  backend execution. The provider must not depend on a Transit SDK, cgo/native
  library, ABI pin, principal coordinator or signed-envelope journal.
  `internal/nativecontract/` and native-only live/release checks are superseded
  scaffolding pending removal/replacement under the milestones 10–17 HTTP plan;
  historical qualification records do not authorize extending that architecture.
  Keep test-only service
  behavior in `internal/testutil/mockplatform/`, contract fixtures in
  `testdata/contracts/`, examples in `examples/`, and generated docs in `docs/`.
- Use the published platform management contract where available. Until it
  exists, develop against the plan's versioned, explicitly provisional test
  contract and stateful HTTP mock. Exercise the actual provider/client code;
  keep mock-only resources experimental until qualified against the real API.
  Use Go `net/http` and schema-derived boundary types. Exact routes, authentication,
  API versions, idempotency and operation lookup belong to Platform's public HTTP
  contract. Missing HTTP semantics are Platform prerequisites, not permission to
  expose its Transit wire protocol or adopt a native client in this provider.
- Keep plan, refresh, and data-source reads observational. Do not provision
  services by writing private platform tables or invoking cloud CLIs, SQL,
  migrations, or Nix from the provider.
- Use stable scoped IDs, explicit target selection, verified TLS, bounded
  timeouts, and revision preconditions. Distinguish accepted operations from
  completed work. Establish recoverable request identity before mutation;
  uncertain retries must inspect or resume the original operation through the
  HTTP API, including after death before Terraform saved a resource/operation ID.
  Require server-backed durable acceptance, ownership and generation/revision
  conflicts; private Terraform state alone cannot supply pre-dispatch persistence.
- Preserve principal-bound ownership and durable receipt tombstones. Import
  cannot transfer ownership; identical credentials and keys cannot distinguish
  cloned states from recovery. Parent loss and incomplete discovery cannot prove
  child absence. Keep organization-only bootstrap free of configuration cycles.
- Remove state only on authoritative absence. Denied reads, outages, and
  incomplete listings are errors. Use dependency-restricted deletion and
  explicit data protection; preserve separately owned data and artifacts.
- Keep management settlement separate from controller readiness. Bindings use
  consumer IDs; workloads list required slot names, never binding IDs. Runtime
  access requires explicit service-specific grants. Preserve active revisions
  through failed rollout and retain independent data/artifacts/grants on retirement.
- Use hard cutovers for development contracts until all roadmap milestones 1–17
  are complete. Maintain no backwards compatibility for previous mock contracts,
  fixture ledgers or Terraform state. Update client, fixtures, schemas, tests and
  examples together and remove superseded migrations/fallbacks. Historical
  qualification records are evidence, not compatibility obligations.
- Preserve current-contract ownership, fingerprints, receipts and crash recovery.
  Reject incompatible state/protocol versions before mutation; never automatically
  erase state, reinterpret uncertain work or delete retained services at cutover.
  Keep published artifact versions immutable even while compatibility is deferred.
- Keep credentials and secret values outside Terraform state, plans,
  diagnostics, logs, and the Nix store. A sensitive schema flag alone does not
  prevent values from being stored.
- Preserve the MIT license. Keep examples and generated resource documentation
  aligned with implemented behavior. After schema or example changes, run
  `./scripts/generate-docs.sh` from the Nix shell. It generates into a temporary
  directory and replaces only the schema reference, preserving `docs/roadmap/`.
  Do not run tfplugindocs directly against `docs/`; it cleans its output directory.

## Validation and CI

Run `just fmt`, `just check`, and `just build` before handing off code changes.
For Nix changes, also run
`nix flake check --all-systems --no-build --no-update-lock-file`; evaluation on
other systems does not establish that their binaries build or run.

Add focused tests for changed provider behavior, including null/unknown values,
import, drift, authorization failures, revision conflicts, protected deletion,
and interrupted operations where relevant. Real lifecycle acceptance requires
an explicitly configured isolated backend; ordinary checks must run without
service credentials, production state initialization, or provisioning effects.

Use the installed-binary harness for configurations with provider aliases:
reattaching one in-process server shares its configuration state across aliases.
Keep fixture transitions deterministic and persist synthetic acceptance before
notifying crash barriers. Distinguish killing the separate provider process from
restarting the fixture service and reloading its ledger in the test harness.

`.github/workflows/ci.yml` runs the check and build tasks on Linux, including
mock acceptance and Linux process-crash recovery. Keep GitHub Actions pinned to
verified commit SHAs with
version comments, read-only repository permissions, and no persisted checkout
credentials. Keep real acceptance tests outside this ordinary CI workflow.

Summarize what changed, which checks ran, and any skipped or unqualified behavior.
Do not claim cross-platform or real-service support from local unit tests alone.

## Distribution and live gates

- Package Linux amd64 only until additional systems execute the qualification
  matrix. Four-system shell evaluation is not package/runtime support.
- Keep `VERSION` and commit-derived immutable identities consistent. A dirty
  snapshot cannot be published. Build the release ZIP from exact mirror bytes,
  never a second compiler invocation. Keep signing keys outside Nix.
- The independent consumer deliberately pins a committed package revision. Update
  its flake, exact provider constraint and both locks together, then execute a copy
  outside the checkout. Never bypass readonly checksum checks.
- `just test-acc installed` injects a mirror/version into the existing process
  harness; missing artifacts fail without a Go build fallback. Its Terraform
  processes must have no compiler in PATH.
- Keep release/live workflows separate from synthetic CI. Full real service
  evidence gates publication; metadata-only, mocked, missing or skipped cases
  cannot satisfy it. Record concrete upstream blockers without fabricating APIs.
