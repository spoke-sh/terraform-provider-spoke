# 11. Compatibility and release distribution

[Roadmap](README.md) · **Status:** In Progress

Publish an independently versioned provider that a clean consumer can install
and use with an explicit Terraform, platform API and operating-system matrix.

See the [remaining implementation plan](milestones-10-17-implementation-plan.md#milestone-11-current-profile-qualification-and-publication)
for the current HTTP API matrix, Go artifact portability, artifact identity and
publication gates. The [hard-cutover policy](README.md#development-compatibility-policy)
defers backwards compatibility until all milestones 1–17 are complete.

## Prerequisites and scope

Use [Nix/installed consumers](09-nix-packages-and-consumers.md) and
[live qualification](10-live-platform-qualification.md). Compatibility and
artifact automation can be implemented earlier, but this milestone closes only
with executed publication and installation evidence. Additional services in
phase IV are not required for an experimental initial release, but completing
the full roadmap remains the gate for introducing compatibility commitments.

## Delivered foundation and remaining gates

The Linux amd64 / Terraform 1.15.3 / mock-v5 matrix, immutable commit versions,
exact-mirror ZIP builder, protocol manifest, checksums/provenance and gated release
workflow are implemented. Candidate verification and negative publication tests
pass. Signing is outside Nix; retries reject differing existing assets.

No public release, registered signing key, anonymous Registry installation or
installed real-service journey is qualified. Full live evidence and clean-host
Go artifact execution remain prerequisites. The native-only publication checks
still need replacement with HTTP contract and advertised-capability evidence
checks; that implementation is pending. See [distribution](../distribution.md)
and the [current execution record](../../qualification/milestones-10-17.md).
The earlier [mock-v4 record](../../qualification/milestones-9-11.md) remains historical.
The current expanded artifact registers the later service families, so its
publication gate requires their real evidence too. A smaller initial release
must explicitly narrow its advertised surface; partial evidence cannot qualify
this artifact.

## Delivery sequence

1. Select and execute the minimum Terraform version deliberately. The existing
   executed baseline is 1.15.3; the earlier 1.10 candidate is historical and does
   not impose an older-CLI support requirement. Test the chosen
   minimum and pinned current version with protocol 6, validation, lifecycle,
   import, saved plans and current-contract recovery before claiming a range.
2. Run the Go provider on Linux amd64/arm64 and macOS amd64/arm64 before
   advertising those targets. Port process-death tests to an appropriate system
   harness where needed; the Linux `/proc` test does not qualify macOS recovery.
   Record unavailable targets explicitly instead of treating cross-builds as proof.
3. Define independent prerelease identity and the supported current API/state
   profile. Until all milestones 1–17 are complete, use hard cutovers without
   historical mock/state migrations. Test current-contract state/recovery and
   incompatible-version rejection before mutation; never reinterpret accepted
   work or discard state to make a cutover succeed.
4. Build platform ZIPs, checksums, signed checksum metadata, protocol manifest,
   source provenance and generated docs. Test the extracted Go artifact on a
   clean host without Nix, compilers or Transit libraries. Replace native-only
   publication profile/pin checks with the published HTTP schema, Platform revision,
   exact artifact and full advertised-family evidence, retaining mock/partial-report
   rejection. No native SDK/ABI closure is a provider release prerequisite. Pin
   workflow actions, minimize job permissions and gate publication on the selected checks and exact tag source.
5. Confirm namespace/Registry ownership and publication requirements when
   implementing the pipeline. Publish immutable artifacts and matching mirrors;
   a partial failure is visible and retryable without replacing published bytes.
6. Verify anonymous release/Registry downloads from a clean external consumer,
   checked lockfiles, install/reinstall, full graph onboarding and the scoped live
   lifecycle. Publish supported operations, versions and known limits together.

## Acceptance criteria

- The published compatibility matrix names executed CLI/API/system combinations
  and current-contract state behavior. No historical compatibility is promised
  before the full-roadmap gate. Unsupported combinations fail predictably;
  evaluation or compile-only results remain labeled separately.
- Independent version metadata, tag, artifacts, checksums, manifest, docs and
  Nix mirror agree. Identical version/platform identities never carry differing
  provider bytes. Credentials are absent from archives and provenance.
- Release automation passes on a qualified committed revision and real public
  artifacts are anonymously installable. A pipeline definition alone is not
  completion evidence.
- A consumer without a compiler or provider checkout initializes with checksum
  enforcement and completes the documented installed/live journey. Public
  availability and authenticated service access are tested separately.
- Release and recovery instructions describe immutable rollback/version
  selection, API compatibility and hard-cutover constraints without promising
  that changing provider binaries rolls back accepted platform operations.
