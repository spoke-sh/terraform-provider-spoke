# 18. Static application and asset publication

[Roadmap](README.md) · **Status:** In Progress

Declare applications, publication targets/configuration and domains through a
Platform-owned API, with import, drift and recoverable lifecycle. Terraform
resource names and schemas remain undecided; no static resource is registered.

## Delivered client foundation, 2026-11-02

[Platform PR #40](https://github.com/spoke-sh/platform/pull/40) implements a
separate declaration HTTP contract over its retained native publication owner.
The provider now pins its `publication-v1` schema and supplies a typed HTTP client
for project applications/bindings, environment targets and original-operation
discovery. Decimal revision strings, per-mutation keys and opaque fingerprints
remain independent of the bucket `mock-v5` management-key algorithm.

The [execution record](../../qualification/publication-client.md) attributes
passing real local HTTP client qualification to exact clean provider/Platform
revisions. It covers declaration lifecycle and replay, drift/stale revisions,
scoped denial, protected/dependency deletion, paginated history and recovery after
actual caller process death. The normal profile rejects synthetic peer behavior;
there is no upload, selected-head mutation, real object service or hosted endpoint.
The upstream API was still an open candidate at review; its separate CI must
establish its own result. No Terraform static resource is registered here.

[Consumer issue #13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13)
tracks Saga's nightly/production acceptance, including an exact immutable Nix
mirror independently of the full Registry release gate. This client work starts
TF18; it does not complete installed lifecycle or static delivery acceptance.

## Prerequisites and delivery

Consume the HTTP/recovery boundary in [milestone 10](10-live-platform-qualification.md),
scope/access semantics in [milestone 4](04-scope-and-access.md) and real object
storage from [milestone 6](06-databases-and-object-storage.md). The implemented
operator-provisioned scopes/roles can qualify a bounded declaration adapter before
general customer bootstrap. Actual immutable delivery still requires Platform's
P6 service; completed provider mocks cannot satisfy it. Platform P7 owns
the [proposed static-publication contract](https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/static-publication-v1.md)
and [tracker #29](https://github.com/spoke-sh/platform/issues/29).

1. Agree application/target identities, field ownership, audience, quotas,
   immutable bundle references and expected-revision fences. Decide explicitly
   whether selected heads/promotions are declarative or separately owned runtime
   operations. Domain integration consumes only the qualified static HTTPS slice;
   it need not wait for milestone 15's private connectivity.
2. The declaration schema/client foundation is pinned and locally qualified.
   Next define the Terraform lifecycle adapter's resource ownership/import model
   and stable per-intent request identity before dispatch. Cover create/update/
   delete, A → B → A, replacement and death before state save; unsaved random keys
   and the bucket generation algorithm cannot supply this contract. Adapt Terraform,
   fixtures, examples and generated documentation together. Keep build/upload/
   sealing and per-file bytes outside Terraform state. Prebuilt delivery does not
   depend on hosted Nix builds or general containers.
3. Exercise validate/plan/apply/read/import, empty plans, external drift, denied
   access, stale revisions, uncertainty before state save and protected deletion.
   GET/plan/refresh must not upload, promote or advance work. Removing declarations
   must preserve separately owned bundles, data and external DNS.
4. Qualify installed lifecycle against the real API and independently fetch
   HTML/JS/Wasm/assets through preview and stable domains. Check audience/cache
   isolation, stale-effect fencing, TLS, warm-cache rollback and open-page assets.

## Acceptance and limits

Complete only with exact artifact/API/schema evidence for management, interrupted
work and independently delivered bytes. Metadata CRUD is not static hosting.
Missing APIs remain Platform prerequisites; do not represent this as bucket or
OCI registry resources. Extend release required cases only with the actual new
registered surface. This milestone does not alter TF10's existing graph or the
original 1–17 development compatibility gate.
