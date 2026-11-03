# Implementation plan: GitHub runner pools and build caches

**Status: Proposed; contract and measured delivery plan only.** No provider
schema, mock endpoint, live runner or billable service is added by this plan.
The resource family extends [jobs and schedules](12-jobs-and-schedules.md), using
[artifact identity](05-artifact-registries.md) and
[capacity field ownership](17-scaling-and-placement.md). Existing completion
records remain unchanged; there is no second milestone sequence.

## Customer outcome and service owner

A customer declares GitHub Actions runner capacity and cache policy in Terraform,
reviews a plan, and directs workflows to the returned routing selector. GitHub
owns workflow scheduling and job attempts. Platform provisions isolated runners,
enforces budgets, reconciles accepted effects and meters use. This provider is
only a Go client of the public Platform HTTPS contract.

The proposed service contract lives in
[Platform's runner-pool contract](https://github.com/spoke-sh/platform/blob/563e0eeab3477c329a457227db436d5eca70d251/contracts/github-runner-pool-v1.md),
with a [delivery plan](https://github.com/spoke-sh/platform/blob/563e0eeab3477c329a457227db436d5eca70d251/docs/roadmap/ci-runner-implementation.md)
and [measured Transit baseline](https://github.com/spoke-sh/platform/blob/563e0eeab3477c329a457227db436d5eca70d251/docs/qualification/ci-runner-baseline.md).
These links pin the proposed Platform documentation at `563e0ee`; reconcile
the merged contract revision before implementation. They are not executable API dependencies.
The service work is tracked in [Platform #56](https://github.com/spoke-sh/platform/issues/56);
provider follow-up stays in [implementation tracker #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).

## Proposed Terraform surface

| Surface | Desired state or observation |
| --- | --- |
| `spoke_github_runner_pool` | Environment, GitHub connection reference, repository IDs/workflow policy, execution profile, CPU/memory/scratch requirements, min-idle/max-concurrent, deadlines, cache reference, budget policy and drain policy |
| `spoke_build_cache` | Environment, `nix-binary-cache-v1` protocol, quota/retention and authorized ownership; server-managed signing/credentials |
| Exact/collection pool data sources | Stable IDs, desired/active revisions, routing selector, readiness/blockers and bounded capacity observations |
| Exact/collection cache data sources | Identity, non-secret endpoint/trust metadata, protocol and readiness; no credentials or private object listing |

Names and fields remain proposed until the versioned HTTP schema is reviewed.
Publish supported profile/architecture/shape combinations through capabilities;
reject unsupported input rather than synthesize a VM configuration in the client.
Import uses stable IDs and current authority. Plan and refresh cannot register a
runner, start a job, build an artifact, upload a cache or repair a cloud resource.

GitHub installation consent creates an authorized Platform connection separately.
Terraform consumes its ID; never accept App private keys, installation tokens,
JIT configuration or AWS credentials as resource attributes. Marking a Terraform
field sensitive would still store it and is insufficient.

No Terraform resources represent individual runner VMs or GitHub job attempts.
Their IDs/counts/results are observations owned by Platform/GitHub. Pool updates
change future admission and drain obsolete profiles. Delete disables admission
then waits for bounded management settlement; active/uncertain work can block
retirement. Deleting a pool preserves its independently owned cache and artifacts.

## Delivery changes

1. **Pin the Platform HTTP profile.** Agree exact routes, schema, fingerprints,
   current authorization, opaque IDs, expected revisions, bounded discovery and
   capability refusals. Publish canonical vectors before copying the schema.
   Establish durable management identity before mutation, including provider
   death before an operation/resource ID reaches Terraform state. Existing
   bucket/publication profiles do not implicitly cover these resources.
2. **Implement deterministic fixtures.** Model accepted policy versus readiness,
   virtual GitHub/cloud observations, capacity reservations, duplicate/reordered
   notifications, uncertain launch/stop, budgets and protected cache deletion.
   Keep the fixture test-only. Mock scheduling, usage or cache hits do not establish
   a real runner, performance improvement or customer charges.
3. **Implement the HTTP client and resource adapters.** Use the shared provider
   transport boundary, unknown/null handling, explicit update/drain semantics,
   state recovery and observational data sources. Preserve runtime-owned fields;
   Terraform must not reset actual instance counts on refresh/apply. The pool
   owns min/max intent initially; a separate scaling policy cannot also own it.
4. **Generate documentation and package the provider.** Add examples only after
   the schema exists. Run the existing documentation generator, boundary audit,
   unit/race and mock acceptance suites; build the exact immutable Linux amd64
   artifact and update an independent consumer's provider/mirror checksums.
5. **Qualify installed lifecycle.** Execute the real Terraform binary with two
   independent scopes and a retained Platform API. Cover create/update/delete,
   import, empty-plan, denied/incomplete reads, stale revisions, provider death
   before/after acceptance and state save, service restart and drain/cancellation.
6. **Qualify real execution separately.** One declared pool must register a real
   ephemeral runner, run an allowed GitHub job, preserve its artifact/usage
   identity and terminate the VM. Cover orphan/duplicate events, revoked
   installation, budget exhaustion, denied cross-tenant cache access and a
   broken control-plane upgrade with external recovery. Only then advertise
   that specific hosted profile. Paid service needs Platform billing gates too.

## Platform dependencies and next proof

The reviewed provider is `41024c6b72712e4cc5203e9ff9c07269036eed4f`; Platform is
`8c494e469eec34de5d3c74a4286f1ca5cb02a833`, reviewed 2026-11-03. The existing
provider still has 26 resources/53 data sources and no runner/cache family.
Native Transit and API/provider pins are unchanged. Platform P11 owns build/cache
trust, P12 isolated worker recovery, P19 bounded pool capacity and P9/P10 paid
usage. P4/P5 supply management and identity; an internal pilot can use bounded
slices without waiting for unrelated resource families.

The next provider task is contract/vector review, then a fixture/client change
only after Platform publishes the schema. The first performance task belongs
to Transit: measure setup, compilation and tests separately and reuse exact
build outputs. A declarative provider resource cannot itself make tests faster.

The existing TF12 job/schedule mock implementation and TF17 scaling mock evidence
are preserved. This new externally scheduled finite-job profile and its cache
are unfinished TF12 scope; it does not reopen TF5's completed OCI mock scope or
pretend its registry protocol is a Nix cache. TF17 retains general scaling and
placement qualification. Reviewed ordering stays unchanged because the existing
scope/artifact/access contracts precede these still unfinished execution families.

## Required evidence before release claims

- No native/Transit, GitHub or AWS SDK/network path outside the public Platform
  HTTP client; no worker starts during plan/refresh/discovery.
- Original-key management recovery across provider/service deaths; no double
  launch or billing identity created by a retry.
- Exact source/schema/provider artifact and exercised HTTP profile in the
  independent consumer record; matching API capability evidence.
- Real job assignment, cancellation/expiry, retained logs/artifacts and physical
  termination; management acceptance alone cannot establish readiness.
- Current authorization, tenant isolation, protected cache deletion and one
  owner for capacity fields, with secret canaries absent from Terraform state.
- Physical cost and customer usage remain distinct; failed/unknown outcomes and
  duplicate notifications reconcile under Platform's meter/rate policy.

The first pilot is internal, Linux/x86_64 and non-spendable. ARM64, arbitrary PR
code, shared hosts and paid customers require their own service evidence. The
100× ambition is a measured build-reuse hypothesis, not a supported performance
claim or an acceptance test that can be satisfied by skipping qualification.
