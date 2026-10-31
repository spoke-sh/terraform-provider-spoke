# 11. Compatibility and release distribution

[Roadmap](README.md) · **Status:** Planned

Publish an independently versioned provider that a clean consumer can install
and use with an explicit Terraform, platform API and operating-system matrix.

## Prerequisites and scope

Use [Nix/installed consumers](09-nix-packages-and-consumers.md) and
[live qualification](10-live-platform-qualification.md). Compatibility and
artifact automation can be implemented earlier, but this milestone closes only
with executed publication and installation evidence. Additional services in
phase IV are not required for this initial release.

## Delivery sequence

1. Select and execute the minimum Terraform version deliberately. The original
   candidate is 1.10; the existing executed baseline is 1.15.3. Test the chosen
   minimum and pinned current version with protocol 6, validation, lifecycle,
   import, saved plans and upgrade behavior before claiming a range.
2. Run native Linux amd64/arm64 and macOS amd64/arm64 qualification before
   advertising those targets. Port process-death tests to an appropriate native
   harness where needed; the Linux `/proc` test does not qualify macOS recovery.
   Record unavailable targets explicitly instead of treating cross-builds as proof.
3. Define independent SemVer, prerelease identity, API compatibility and resource
   state schema upgrade policy. Test existing states and provider upgrades;
   reject incompatible versions before mutation and never silently reinterpret
   accepted work or discard state to make an upgrade succeed.
4. Build platform ZIPs, checksums, signed checksum metadata, protocol manifest,
   source provenance and generated docs. Pin workflow actions, minimize job
   permissions and gate publication on the selected checks and exact tag source.
5. Confirm namespace/Registry ownership and publication requirements when
   implementing the pipeline. Publish immutable artifacts and matching mirrors;
   a partial failure is visible and retryable without replacing published bytes.
6. Verify anonymous release/Registry downloads from a clean external consumer,
   checked lockfiles, install/upgrade, full graph onboarding and the scoped live
   lifecycle. Publish supported operations, versions and known limits together.

## Acceptance criteria

- The published compatibility matrix names executed CLI/API/system combinations
  and state upgrade paths. Unsupported combinations fail predictably; evaluation
  or compile-only results remain labeled separately.
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
  selection, API compatibility and state upgrade constraints without promising
  that changing provider binaries rolls back accepted platform operations.
