# Implementation plan: CI supply governor

**Status: Proposed; contract and measured delivery plan only.** No provider schema,
mock endpoint, live runner or billable service is added. This extends
[scaling and placement](17-scaling-and-placement.md), using existing
[scope/access](04-scope-and-access.md) and [artifact identity](05-artifact-registries.md).
[Jobs and schedules](12-jobs-and-schedules.md) retain their existing execution
semantics and evidence; no new customer CI resource is required there.

## Customer demand and operator supply

Customers use the workflow integration in
[`spoke-github-action`](https://github.com/spoke-sh/spoke-github-action). Their
repositories declare commands and optional runtime compatibility, with no capacity
Terraform, pool/cache provisioning, fleet count, machine size or governor ID.
GitHub produces queued-job demand and owns scheduling/attempts. Platform resolves
the authorized account and governs eligible supply, managed cache access and use.

A separately authorized operator uses this provider in an infrastructure root to
manage `spoke_ci_governor`. Terraform declares limits and policy, not demand or
actual worker counts. Only Platform translates that policy into physical supply.
The provider remains a Go client of the public Platform HTTPS API.

The [Platform demand/governor contract](https://github.com/spoke-sh/platform/blob/dcb40932b1e927b0525a3d6fa392142ed0b5044a/contracts/ci-demand-and-governor-v1.md)
and [delivery plan](https://github.com/spoke-sh/platform/blob/dcb40932b1e927b0525a3d6fa392142ed0b5044a/docs/roadmap/ci-runner-implementation.md)
own the service behavior. The [historical timing baseline](https://github.com/spoke-sh/platform/blob/563e0eeab3477c329a457227db436d5eca70d251/docs/qualification/ci-runner-baseline.md)
retains its original source attribution. These links pin the refined proposal at `dcb4093`; reconcile the merged
contract revision before implementation; these documents are not an executable API schema. Track service
work in [Platform #56](https://github.com/spoke-sh/platform/issues/56) and provider
work in [implementation tracker #1](https://github.com/spoke-sh/terraform-provider-spoke/issues/1).

## Proposed Terraform surface

| Surface | Policy or observation |
| --- | --- |
| `spoke_ci_governor` | Operator environment, admission/budget/managed-cache policy references, qualified execution profiles, active-job/vCPU/warm-capacity ceilings, deadlines and drain policy |
| Exact/collection governor data sources | Scoped stable identity, desired/active policy revisions, supported profiles, readiness/blockers and bounded observations |
| Demand, workers and usage | Platform/GitHub observations; never Terraform resources or desired-count fields |

The earlier proposed `spoke_github_runner_pool` and required `spoke_build_cache`
customer resources are removed from this slice. Managed cache selection is a
server policy; cache lifetime remains independent of governor/worker retirement.
No customer Terraform is needed to use the workflow.

The governor is the sole owner of its supply policy. Reject a generic scaling
resource attempting to own the same fields. Enforce current operator authority,
opaque scoped IDs, expected revisions and advertised profile combinations. Import
cannot transfer authority. Plan, refresh and discovery cannot generate demand,
register a runner, start a job, write cache data or repair cloud resources.

GitHub installation consent is separate onboarding. Never accept App keys,
installation/JIT tokens, AWS credentials or worker secrets as resource fields;
sensitive Terraform attributes still store values. Customer job credentials must
not mutate a governor or select another account's supply authority.

Lowering ceilings changes new admission and drains surplus; it does not silently
kill accepted work. Raising limits with zero demand launches nothing. Deletion
stops admission and drains; unresolved leases/usage can block completion. Preserve
receipts, tombstones and caches. Successor-governor handoff must fence the old
owner without duplicating reservations. Actual worker counts remain observations.

## Reviewable delivery changes

1. **Pin the public governor HTTP profile.** Agree schema, routes, canonical
   intent/fingerprint vectors, operator authorization, revisions, bounded reads
   and capability refusals. Establish recoverable management identity before
   dispatch, including provider death before saving any ID to Terraform state.
2. **Implement deterministic fixtures.** Model global and tenant/account ceilings,
   multiple repositories sharing one allowance, queued/refused/expired demand,
   duplicate notifications, stale observations and uncertain launch/stop. Include
   zero-demand apply, lowered limits, overlapping governors and handoff fencing.
   Mocks cannot qualify GitHub routing, fleet isolation or billing.
3. **Implement HTTP client and governor adapters.** Cover unknown/null handling,
   original-key recovery and explicit drain semantics. Preserve runtime-owned
   counts and account-owned budget policy; no private Transit/GitHub/cloud clients.
4. **Document and package the exact provider.** Generate supported examples only
   after the schema exists; use the existing boundary audit, unit/race, mock
   acceptance and immutable installed-artifact workflow. Examples belong to an
   operator infrastructure root, never the customer workflow repository.
5. **Qualify installed management.** Run independent Terraform scopes with actual
   service retention: create/update/import/empty-plan/delete, authorization,
   stale revisions, provider death before/after acceptance/state save, service
   restart, policy reduction, drain and preserved cache/usage evidence.
6. **Qualify workflow demand against governed supply.** A customer repository with
   no Terraform calls the reusable workflow. Observe actual GitHub demand, one
   admitted job/lease, logs/artifacts and physical termination. Test a second
   repository sharing quota, cross-tenant refusal, cancellation, GitHub outage,
   orphan registrations and broken-control-plane recovery. Change supply policy
   without changing the consuming repository. Paid service still needs billing
   gates; cross-organization access requires independent qualification.

## Roadmap and next proof

Provider main is `41024c6b72712e4cc5203e9ff9c07269036eed4f`; reviewed Platform main
is `89875815497e4e29b4fd4b035faa318f2ba4d857` on 2026-11-03. The current provider
still registers 26 resources/53 data sources and no governor. Native/API/artifact
pins remain unchanged. TF17 owns the proposed operator governor; TF12's original
job/schedule obligations and TF5's completed OCI mock scope remain unchanged.
This corrects the earlier plan's TF12 pool/cache extension; no milestone status
or sequence changes. Existing scope, artifact and lifecycle contracts precede
the governor. Its narrow execution profile does not require all generic services.

Platform P11 owns image/cache trust, P12 worker isolation/recovery, P19 supply
policy and P9/P10 paid usage. P4/P5 own management and identity. The workflow repo
owns customer integration and distribution, not supply policy or cloud effects.
Next provider action: review the governor schema/vectors alongside the workflow
contract, then fixture/client work. Transit can separately improve measured build
reuse now; declaring a governor does not itself accelerate compilation or tests.

## Required evidence before release claims

- No worker or demand from plan/apply without queued work in the zero-warm profile.
- One policy owner and atomic operator/tenant limits across repositories;
  uncertain workers keep reservations until proven stopped.
- Original-key management recovery across provider/service death, with no double
  launch identity or settlement caused by retry.
- Exact provider artifact/schema/service profile in the independent consumer;
  real workflow routing, cancellation/expiry, logs/artifacts and physical cleanup.
- Customer jobs cannot mutate supply policy or impersonate another account;
  secret canaries stay out of state and caches remain isolated.
- Physical cost and billable job usage remain distinct; operator warming is not
  billed as customer execution. Paid rates/reservations require separate proof.

The first pilot is internal Linux/x86_64 with non-spendable usage. ARM64, public
PRs, cross-organization callers and paid customers retain separate gates. The
100× ambition remains a measured build-reuse hypothesis, not a release claim.
