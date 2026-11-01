# Milestones 9–11 execution record

> **Planning correction, 2026-11-01:** the
> [Platform HTTP plan](../docs/roadmap/milestones-10-17-implementation-plan.md)
> supersedes this record's native SDK/ABI, signer, principal-sequence journal
> and native-runtime release prerequisites. Platform owns Transit integration;
> this provider remains a Go HTTP client. Results below are preserved for their
> original revisions and scopes. Existing native-only live/release checks still
> need implementation changes; this note supplies no new live evidence or status.

Executed 2026-10-31 on Linux amd64 with locked Go 1.26.7, Terraform 1.15.3 and
protocol 6. Milestone **9 is completed**. Milestones **10 and 11 remain in
progress**, with native codecs/live prerequisites and release construction
implemented; no native service or public release is qualified.

## Hard cutover

`5738145` replaces mock-v3 with mock-v4 and resource schema version 2. Previous
state/ledger migrations, mock-v1/v2 specifications and captured migration corpora
were removed. All intents require an explicit resource kind; bucket storage and
status are mandatory. Fingerprints use the mock-v4 domain separator. Prior state,
wire shapes, routes, ledger versions and incompatible stored fingerprints fail.
Current ownership, receipts, retained data and crash recovery remain covered.
Historical milestone evidence links to the removed files at their original commit.

The first full run found a test expecting the wrong spelling of Terraform's
state-upgrade diagnostic; Terraform had correctly rejected the old state. The
assertion was corrected, the focused test passed, and subsequent artifact runs
include the rejection check. No compatibility fallback was restored.

## Nix and independent installation

Exact source `5708eb1dad834acf04d4c6f2b654a22af25696d9`:

- Version `0.1.0-dev.g5708eb1dad834acf04d4c6f2b654a22af25696d9`.
- Binary SHA-256 `1510b2f80108abc66e5806478fe4d4c832ab804d49a732ab14a78410f308091e`.
- Mirror `/nix/store/m0gkqyy8k78ny1hq4bhm2sap442n4rlq-terraform-provider-spoke-mirror-0.1.0-dev.g5708eb1dad834acf04d4c6f2b654a22af25696d9`.
- Verified vendor hash `sha256-9a4TDOtr/IVrV/zHl1eQLGrEomh/9Vl4lBJ1iC2Ha8c=`.

The full `just test-acc installed` run passed **33 top-level mock acceptance
tests** in 347.762 seconds, including bucket CRUD/import/empty plans, alias and
owner isolation, the complete service graph, partial applies, retained-data and
secret canaries, lost responses, fixture-ledger reload and actual separate
provider-process death. The artifact branch verifies version/contract/platform
identity and refuses to build from source on a missing or mismatched artifact.
The process-death harness identifies the launched executable through `/proc`.

Normal mirror initialization and readonly reinitialization passed. Tampered
checksums, unavailable versions and missing platform directories failed without
rewriting the trusted lock or escaping Spoke's direct-install exclusion. The Nix
install check also ran schema loading and validation inside its bounded sandbox.

The [independent consumer](../examples/installed-consumer/README.md) was copied to
`/tmp/spoke-independent-consumer`, outside the provider checkout, and passed:
readonly init, bucket creation, empty plan, state removal/import preserving the
original identity, another empty plan and teardown. Its shell rejected Go, Zig
and C compilers on PATH. The separately packaged synthetic TLS fixture created
runtime CA/credential/ledger files outside Nix. HashiCorp Random 3.7.2 installed
normally alongside the Spoke mirror using its signed origin checksums.
The initial missing Spoke lock entry caused the intended readonly failure; the
lock was then deliberately generated from the exact trusted mirror and committed.

All four development shell/formatter systems evaluated with
`nix flake check --all-systems --no-build --no-update-lock-file`. Only Linux amd64
packages and process recovery were built/executed and are advertised.

## Native and live gates

Platform `230c2c4cee12c08aaeaaf65e3c252a6d50450a55` still provides the same restricted
bucket metadata profile. The pinned Transit Go SDK lacks required Serving and
subject-proof interfaces; it cannot stand in for the native management transport.
See the [field-by-field boundary record](native-boundary.md).

Native application commands, precondition-bound fingerprints, read-proof context
bytes and canonical result decoding (including revision-zero pending creates) passed independent Python-generated vectors
and malformed-input checks. These codecs are unregistered and do not sign/send.
No SDK round trip, native adapter, durable signed-operation journal or native
restart recovery is claimed. These depend on the shared upstream boundary.

Explicit live preflight with the exact candidate digest emitted ten blocked
required cases and exited nonzero. The report listed missing shared SDK support,
unqualified native adapter/journal and metadata-only platform services. The
release gate rejected that report. The [captured report](native-preflight.json) records these blocked cases. No
credential was read or network request sent by preflight. Missing/skipped/duplicate cases, empty reports, identity mismatches
and incorrect artifact digests are tested rejection paths. The configured
self-hosted workflow was linted, not executed against a real placement.

## Release candidate

Source `70548da5ab39024d3a651fa2824ef3173ea01376` produced:

- Version `0.1.0-dev.g70548da5ab39024d3a651fa2824ef3173ea01376`.
- Binary SHA-256 `d0a181ab3f9000431169cdc0e9de5a3b228f841faac6768ba22cdac97d4e6722`.
- Mirror `/nix/store/a7aa5nak2w4wgg63rflbqpng4k1s3di0-terraform-provider-spoke-mirror-0.1.0-dev.g70548da5ab39024d3a651fa2824ef3173ea01376`.
- Candidate assets staged at `/tmp/spoke-release-70548da`, with protocol-6
  manifest, ZIP, license, source/input provenance and complete SHA-256 manifest.

The builder/verifier passed using the exact Nix mirror bytes. Archive payloads
match the complete mirror payload; no second binary is built. Deterministic
construction, dirty-source refusal, changed/truncated local archive fixtures, unknown assets,
blocked live evidence and changed existing publication assets passed four Python
unit tests. A temporary **synthetic, unregistered RSA key** signed and verified
the checksum manifest; its private keyring was deleted. This is signing-mechanics
evidence, not a publisher signature or Registry registration.

The ZIP was extracted into an external versioned mirror and all 33 top-level
acceptance tests passed in 342.027 seconds, including the full graph and process
recovery. The later codec-only correction at `ab4a2a3` passed focused race tests;
that unregistered codec is not linked into the qualified mock provider binary. Candidate-only
checks do not establish public availability, a real data plane or future native
runtime portability. The release workflow was validated locally; GitHub signing
and publication were not executed. No tag or release was created.

## Validation and open gates

`just fmt`, `just check` and `just build` passed. The final repository acceptance
run passed all 33 top-level mock tests in 359.284 seconds. Checks included Nix
installation, Go vet, race-enabled unit tests, workflow lint, Terraform/Nix/Go/Just
formatting and the four release-tool tests. The subsequent pending-receipt and
old-header additions passed focused race tests and vet. Both Nix and extracted
ZIP artifact runs passed separately as recorded above.

All-system flake evaluation, 255 local Markdown links/fragments, all 17 roadmap
statuses/numbering, `git diff --check`, and default Just listing passed. Exactly
five public tasks remain. The original main worktree stayed clean.
[Machine-readable artifact evidence](milestones-9-11-artifacts.json) records exact
source, hashes and outcomes. The [distribution guide](../docs/distribution.md)
documents commands, immutable retries, signing and release prerequisites.

Remaining completion gates are explicit:

1. Transit-owned Serving/proof/TLS/native binding interfaces; provider native
   transport and durable pre-dispatch journal with real crash recovery.
2. Platform customer scope/IAM and actual OCI/object/database/secret/workload/TLS
   services, isolated fault/restart controls and the complete retained-data graph.
3. Full live qualification for the exact candidate, native runtime portability,
   namespace/key registration, actual immutable publication and anonymous
   checksum/signature-locked Registry installation with the installed live journey.

Until these pass, milestones 10 and 11 cannot close. Neither a passing mock graph
nor release workflow scaffolding substitutes for those gates. Compatibility with
historical contracts remains deferred until every roadmap milestone is complete.
