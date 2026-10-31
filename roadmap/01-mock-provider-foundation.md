# 1. Mock provider foundation

[Roadmap](README.md) · **Status:** Completed

Let a developer run the first bucket lifecycle through real Terraform and a
local TLS management fixture without platform credentials.

## Prerequisites and scope

This is the bounded foundation merged in
[PR #2](https://github.com/spoke-sh/terraform-provider-spoke/pull/2), not completion
of all contract and qualification requirements in issue #1. The provider,
HTTP client and fixture are separate owners; platform services remain external.

## Delivered foundation

- Go Plugin Framework and protocol 6, explicit endpoint and target scope,
  external credentials, verified TLS and bounded operation timeouts.
- [Provisional mock-v1 contract](../testdata/contracts/README.md), independent
  raw-HTTP/schema checks, a stateful TLS fixture and typed client validation.
- `spoke_bucket` create/read/update/delete/import, exact-ID project/environment
  data sources, stable management keys, generations and revision fences.
- Protected deletion, drift repair, denied-read preservation, lost replies and
  pending operation handling, with an empty second plan after convergence.
- Pinned Nix development tooling, five public Just tasks, generated schemas and
  mandatory mock acceptance in CI.
- A Linux separate-provider-process test kills an installed binary after create
  acceptance, then recovers the same bucket using a checksum-locked filesystem
  mirror and a consumer PATH without Go or development overrides.

## Acceptance evidence

The [merged CI run](https://github.com/spoke-sh/terraform-provider-spoke/actions/runs/41239473685)
passed flake evaluation, formatting, workflow checks, vet, race tests, six mock
Terraform acceptance cases and provider build for branch revision `679cfae`.
The merge revision is `ac0995d1c1f4b205427b5340fce3f0ac560591d0`.

Executable owners are the [Terraform lifecycle tests](../internal/provider/bucket_resource_test.go),
[process test](../internal/provider/process_linux_test.go),
[client tests](../internal/platformclient/client_test.go) and
[raw contract tests](../internal/testutil/mockplatform/server_test.go).
Execution evidence is Linux amd64, Go 1.26.7 and Terraform 1.15.3; evaluation of
four Nix systems is a separate, narrower check.

## Remaining work transferred explicitly

Discovery/capability and ownership contracts remain in [milestone 2](02-management-contracts-and-discovery.md).
The full interruption/isolation matrix and durable fixture restart remain in
[milestone 3](03-recovery-isolation-and-protection.md). The fixture has no OCI,
object or database data plane. It does not survive its own process restart.

The temporary test mirror does not provide Nix packages or a release consumer;
those remain in [milestone 9](09-nix-packages-and-consumers.md). Live qualification
and published distribution remain in [milestones 10](10-live-platform-qualification.md)
and [11](11-compatibility-and-releases.md). Keep this completed scope narrow when
recording subsequent improvements.
