# 9. Nix packages and installed consumers

[Roadmap](README.md) · **Status:** In Progress

Let another repository pin and install the exact provider artifact without
source-tree imports, a compiler in the consumer environment or development overrides.

See the [implementation plan](milestones-9-11-implementation-plan.md#milestone-9-nix-packages-and-independent-consumers)
for package identity, native runtime dependencies, consumer isolation and tests.
The [hard-cutover cleanup](milestones-9-11-implementation-plan.md#step-0-remove-the-compatibility-layer)
precedes new package qualification.

## Prerequisites and scope

Packaging can proceed from [milestone 1](01-mock-provider-foundation.md). Use
its bucket lifecycle as the first installed consumer; include the now-completed
[milestone 8 graph](08-container-services-and-endpoints.md) in this implementation.
This milestone owns package/install mechanics. Publication and final compatibility
claims remain in [milestone 11](11-compatibility-and-releases.md).

## Delivered foundation

The flake supplies pinned development shells. The Linux process test builds a
provider binary, creates a temporary versioned mirror and runs Terraform with
checksum locking and no Go in the consumer PATH. There is no `buildGoModule`
package, exported release mirror or independently pinned downstream fixture.
The refreshed platform profile uses a native SDK/runtime closure; qualify the
changed package again when that dependency is introduced. Development-shell
support does not establish native package or portable release support.

## Delivery sequence

1. Add `packages.<system>.terraform-provider-spoke` using `buildGoModule` and a
   verified dependency hash. Preserve required generated contract inputs in
   source filters. Embed the provider's own version and source provenance.
2. Export a correctly versioned filesystem mirror for
   `registry.terraform.io/spoke-sh/spoke`, with executable layout and immutable
   identities. Evaluation, builds and shell entry must not contact management
   services, deploy resources or place credentials in the Nix store.
3. Define how Nix and release bytes are related. The same published version must
   resolve to the same artifacts/checksums; if source-built bytes differ, use a
   distinct immutable prerelease identity. Never bypass lock verification.
4. Add a consumer outside provider source paths with a pinned flake/release,
   scoped `TF_CLI_CONFIG_FILE` and checked `.terraform.lock.hcl`. Permit normal
   lookup of other providers while excluding Spoke from direct lookup when it
   is mirror-only. Do not use `dev_overrides` as installation evidence.
5. Run init, readonly lockfile re-init, lifecycle and separate-process recovery
   using the Nix-built artifact and an external mock fixture. The packaging job
   may compile; the consuming environment must not compile the provider.

## Acceptance criteria

- Nix builds the advertised package/mirror on every system claimed by this
  milestone, with recorded output identity and dependency hash. Evaluation on
  other systems is reported separately from executable qualification.
- A clean external consumer installs the pinned artifact through Terraform's
  normal version/checksum selection, without provider checkout imports or Go.
- Wrong checksum, unavailable version and unsupported platform fail clearly.
  A readonly lockfile initialization succeeds for the exact tested artifact.
- The installed artifact completes the bucket lifecycle and process recovery;
  the completed milestone 8 graph consumer is mandatory for this milestone
  and the release gate, without blocking the first package-only commit.
- Examples explain credential injection outside the store and source provenance.
  Mirror installation evidence does not claim public Registry availability.
