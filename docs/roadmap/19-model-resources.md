# 19. Model definitions and governed connections

[Roadmap](README.md) · **Status:** Planned

Manage model definitions and external connection references/bindings through
Platform's public management API. Platform owns implicit gateway routing, runtime
invocations, provider adapters, budgets and secret resolution. No model resource
or inference endpoint is implemented by the current provider.

## Prerequisites and delivery

Use the HTTP/recovery boundary in [milestone 10](10-live-platform-qualification.md),
actual scope/access from [milestone 4](04-scope-and-access.md), and the reference/
binding rules in [milestone 7](07-secret-references-and-bindings.md). Platform P14
owns the [proposed model contract](https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/model-resource-v1.md)
and [tracker #26](https://github.com/spoke-sh/platform/issues/26). Complete declared
dependencies by their required slices, not by waiting for unrelated milestones.

1. Agree versioned definitions, scoped connections, model selection, capability
   and budget limits, immutable identities, import and suspension/retirement.
   Assign one writer per field. Model bindings never grant invocation authority.
2. Pin the implemented management schema before choosing Terraform syntax.
   Add typed client/fixtures/schema together, including null/unknown validation,
   bounded discovery, explicit grants and secret references. Provider credentials,
   prompts, transcripts, invocations and tool effects stay outside Terraform state.
3. Qualify CRUD/import/drift, empty plans, denied cross-scope access, revocation,
   revision conflicts, protected retirement and original-operation recovery,
   including provider death before state save. Reads and ordinary lifecycle
   operations must never invoke a model or start paid work.
4. Use the installed artifact against actual management services. Platform must
   separately qualify each selected provider/account/protocol profile's runtime
   inference, budgets, cancellation/unknown outcomes and usage. A tool proposal
   cannot confer execution authority. Finite inference need not wait for hosted
   compute, all adapters, streaming or self-hosted weights.

## Acceptance and limits

Require exact management and installed recovery evidence, independently qualified
runtime adapter claims, secret-canary checks and retained original outcomes.
Missing management APIs remain explicit Platform prerequisites. Expand release
cases only when the corresponding resources are actually registered; keep
unqualified providers and modalities experimental. This milestone leaves TF10's
initial graph and the original 1–17 compatibility gate unchanged.
