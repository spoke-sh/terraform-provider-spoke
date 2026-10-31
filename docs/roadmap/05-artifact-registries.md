# 5. Artifact registries and digest discovery

[Roadmap](README.md) · **Status:** In Progress

Let an application own a private artifact registry and select an immutable image
digest for deployment without running builds or uploads during Terraform reads.

See the [implementation plan](milestones-5-8-implementation-plan.md#milestone-5-artifact-registries-and-digest-discovery)
for contract work, delivery steps and test gates.

## Prerequisites and scope

Use [scope and access](04-scope-and-access.md) and the shared contract/recovery
gates. Platform owns registry provisioning, data-plane authorization, quotas and
retention. This provider manages service configuration and artifact discovery;
the mock does not implement an OCI registry.

## Delivery sequence

1. Specify registry/repository identity, supported formats, retention, quota,
   grants, asynchronous readiness and dependency-safe deletion in the capability
   matrix. Distinguish mutable tags from immutable artifact digests.
2. Add raw contract fixtures and stateful service behavior, including retained
   artifacts, quota errors, rejected access and in-use digest references.
3. Implement `spoke_artifact_registry`, scoped reads and digest discovery with
   revision preconditions and recovery. Refuse ambiguous artifact selection.
4. Document the application workflow: build from pinned source/Nix inputs,
   explicitly upload or promote, verify digest, then apply that digest to the
   workload. A Nix store path is not a remotely deployable image.
5. Keep registry tokens outside provider state and outputs. Define deletion and
   retention changes so service retirement cannot silently purge artifacts used
   by another workload or owner.

## Acceptance criteria

- Terraform lifecycle, import, drift, empty-plan and interruption tests inspect
  both provider state and exact fixture requests/operations.
- Digest discovery is observational, scope-checked and bounded. A tag change does
  not mutate an already pinned deployment; missing and denied artifacts differ.
- Quota, retention, dependency and authorization failures preserve known state.
  Registry deletion refuses retained/in-use content under its declared policy.
- Examples keep build/push outside Terraform resource lifecycle and use explicit
  scoped grants. No upload, build or credential issuance occurs during plan.
- Real OCI push/pull remains an explicit acceptance item in
  [live qualification](10-live-platform-qualification.md); mock metadata alone
  does not establish registry functionality.
