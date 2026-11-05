# CI governor installed Terraform qualification

The experimental `spoke_ci_governor` resource and exact/collection data sources
are implemented against the separately pinned provisional `ci-governor-v1`
profile. Five lifecycle groups and twelve actual provider-death schedules pass
with the exact Nix artifact against an independent, persisted synthetic HTTPS
peer. This is installed Terraform evidence, separate from the historical
[client conformance](governor-client.md). It qualifies no native Platform service,
physical worker, warming, handoff, cache trust, paid usage or hosted endpoint.

## Exact inputs

| Input | Identity |
| --- | --- |
| Implementation and qualification harness | `f0f3cb928e1932fb568f3ae0a9a146d49f1c938e` |
| Artifact version | `0.1.0-dev.gf0f3cb928e1932fb568f3ae0a9a146d49f1c938e` |
| Executable SHA-256 | `b32d905c50f5fea81034c7160d6d8f75788bb2ef36a1a2be4d42612f48eb8cb5` |
| Platform contract source | `af60b8a13d7b9f953010552096e33174c22d3981` |
| Schema SHA-256 | `7899392a4553f748606a59bbccdce03b61edcb93ca6b101d6bf1a192114b15cd` |
| Vectors SHA-256 | `4c5a7b1732d245ae935bc4e996aee1f000a9832eddd3c629c16d490349b4c260` |
| Profile text SHA-256 | `f2b4561ed1e462320596172b9663cb56f7b1d162f8d2157c2244e713cba53cf1` |
| Runtime | Linux amd64, Go 1.26.7, Terraform 1.15.3, locked Nix shell |
| Registered inventory | 27 resources and 55 data sources, all covered by raw HTTPS denial audits |

The installed command runs from a separate clean checkout at the implementation
revision. The main checkout prepares documentation while its full check runs;
all implementation, harness and packaging inputs match the same commit. No
bucket/publication/Saga consumer pin, lock or independent qualification changes.
No real service credentials are used. Four-system shell evaluation is not
cross-platform binary/runtime support.

## Required evidence

The [machine-readable report](governor-terraform.json) records exact commands,
case results, log digests and artifact identity. The
[required-case inventory](../testdata/contracts/governor-terraform-cases.json)
is checked against executable test names. Both existing mock/installed tasks
require governor lifecycle and process-recovery cases; missing artifacts cannot
fall back to a source build in the installed lane.

| Group | Executed boundary |
| --- | --- |
| Lifecycle/import/drift/discovery | Create, update, A → B → A, scoped import, empty plan, observational refresh, revision drift, exact/collection discovery, retirement and new-key recreation. |
| Identity/concurrency | Stale saved revisions fail; changed intent cannot commit a prepared request; matching work resumes the original sequence. Independent roots test identical and conflicting concurrent requests. Cloned credentials/keys remain indistinguishable from recovery. |
| Authority/aliases | Revocation and denied refresh retain state; read-only and foreign principals cannot mutate/adopt ownership. Two installed provider aliases use independent TLS peers and configuration. |
| Protection/drain/observations | Protection requires a completed update. Raised ceilings with zero demand launch no fixture work; lowering limits preserves outstanding usage. Accepted drain times out with state and receipt retained. GET does not settle it; explicit fixture settlement preserves independent references. |
| Schema/confidentiality | Nulls, unsupported profiles and bounds fail; a Terraform built-in data resource exercises unknown-to-known planning. Credential canaries are absent from state, decoded saved plans and captured diagnostics. Terraform subprocess PATH contains its executable directory, with no compiler. |
| Process/peer recovery | Create/update/retire each die before prepare forwarding, after durable prepare, after accepted commit with settlement held, and after completion before the response/state save. All twelve preserve the original tuple, receipt and at-most-once policy effect. |

The crash harness logs the killed provider PID and executable while Terraform
remains alive to report failure. The peer's ledger reload is a separate action
with the same TLS origin. Saved-plan schedules replay the original plan with its
original state copied into a separate consumer; they do not overwrite failed
state or bypass Terraform's serial check. Other schedules recover through a
fresh Terraform plan. Provider death and synthetic ledger reload are neither
native process-crash proof nor power-loss durability qualification.

Additional Framework/selector tests cover private/state-version rejection,
owner/scope/revision/sequence fences, receipt expiry, explicit abort/refusal
successors, capacity rejection, accepted deadlines, malformed/incomplete reads,
revocation before prepare/commit/settlement and bounded complete collections.
The existing client conformance suite separately owns malformed wire details,
cursor loops, snapshot changes, TLS, redirects and canonical vectors.

## Implementation findings and validation

The locked Framework 1.19 drops create private plan data. Creation therefore
freezes owner, sequence and scope-bound fingerprint in computed plan fields
alongside the full configured policy. Updates and null destroy plans use versioned
private metadata. Terraform's ordinary pre-pass before destroy can observe
unchanged live policy during pending retirement; final destroy alone resumes it.
An already retired key is refused before create dispatch, with the diagnostic
deferred past that pre-pass when necessary. See the
[identity protocol](../docs/governor-identity.md) for caller recovery behavior.

Final validation passes: `just fmt`, `just check`, `just build`, documentation
generation, all-system Nix evaluation without builds/lock updates, and
`just test-acc installed`. Ordinary checks include workflow/format validation,
Nix packaging/install checks, Go vet, race unit tests and all 46 mock acceptance
groups (1,227.417 seconds). The installed lane passes all 46 groups in 1,205.047
seconds, including the five governor lifecycle groups and twelve actual provider
deaths; no acceptance group is skipped. Focused Framework, identity, discovery,
failure-retention and raw HTTPS registry tests also pass.

Initial full checks exposed the Nix vendor hash change
caused by newly used list validators from an already locked module; the corrected
pin was committed before the exact artifact run. Initial crash tests also
identified Terraform's destroy pre-pass and post-failure state serial behavior;
both are covered in the final matrix. No dependency versions, toolchain or
current HTTP contract bytes changed.

Platform's [latest review](../docs/roadmap/platform-adoption-implementation-plan.md#governor-adapter-implementation-review-2026-11-04-0648-utc)
records newer build/admission and Stripe work, with current Actions startup
blocked by account payment/spending settings. That is separate from this local
installed evidence. Native governor ownership/cleanup, actual GitHub demand,
isolated workers, service recovery and paid usage remain open under Platform
#56/#20. TF17 stays In Progress; live/release gates remain closed.
