# 18. Static application and asset publication

[Roadmap](README.md) · **Status:** In Progress

Declare applications, publication targets/configuration and domains through a
Platform-owned API, with import, drift and recoverable lifecycle. Terraform
applications, project reservations and environment targets now have experimental
resources and exact/collection discovery. Installed graph qualification and real
delivery remain separate gates.

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
The API and provider client are now merged in Platform #40 and provider #14.
Platform's PR publication-api job passed; the latest merged-main checks are still
running at the [post-merge review](platform-adoption-implementation-plan.md#post-merge-review-2026-11-03-0000-utc).
Historical client evidence retains its original source identity. The subsequent
application adapter uses the [retained-slot protocol](../publication-identity.md);
its [installed Terraform qualification](../../qualification/publication-application.md)
passed at exact provider `581cf627` and Platform `65f8284`.

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

## Next provider delivery sequence

The proposed [Terraform implementation plan](publication-terraform-implementation-plan.md)
expands this sequence into resource schemas, a durable ownership/recovery gate,
logical commits and executable installed-consumer acceptance. Application implementation is in progress; the broader resources wait for its
installed qualification gate.

1. **Application lifecycle and recoverable intent.** Define the Terraform
   request/ownership contract before dispatch: stable create identity, a distinct
   key per logical update/delete, replay of the original intent, import without
   ownership transfer, replacement, A → B → A, expiry and death before state save.
   Prove whether the current API is sufficient; missing durable semantics belong
   in a Platform contract follow-up, not unsaved random keys or a local journal.
   Then implement the application resource and exact discovery/import, wiring the
   publication client into Framework configuration. Exercise actual installed
   create/read/update/delete, empty plans, drift, denial, stale revisions,
   protection and crash recovery before registering broader resources.
2. **Bindings and environment targets.** Add project-owned reservations and
   separately authorized target policy, with exact/collection discovery. Test
   independent project/nightly/production configurations, cancellation against late
   consumption, target death/release, conservative parent holds and safe retry.
   Manage only supported name/protection/retention policy and binding identity;
   runtime heads, bundle upload and promotion remain outside Terraform ownership.
3. **Exact installed consumer.** Pin a committed Linux amd64 Nix artifact, binary
   and schema hashes, exact provider constraint and both consumer locks together.
   Execute a copy outside the checkout without compiler/development overrides.
   Qualify independent roots, scoped credentials/state access and original-operation
   recovery. Local declaration acceptance is distinct from hosted state-backend
   qualification and the eventual Saga domain/byte journey.

Each resource change includes its fixtures, examples, generated docs and installed
acceptance. Keep original receipts and resource IDs across uncertainty; never
reclaim expired keys to work around the current 64-receipt lifetime limit. The
schema/profile bytes already match merged Platform, so a merge alone does not
require a pin cutover. Requalify deliberately when selecting a new fixture/artifact.

Platform can independently implement change C's immutable upload/verification/
sealing. Domains and preview/promotion support are later API gates; do not expose
them through synthetic peer operations. Real HTML/JS/Wasm fetches, isolation and
rollback remain required for completion of this milestone and consumer #13.

## Acceptance and limits

Complete only with exact artifact/API/schema evidence for management, interrupted
work and independently delivered bytes. Metadata CRUD is not static hosting.
Missing APIs remain Platform prerequisites; do not represent this as bucket or
OCI registry resources. Extend release required cases only with the actual new
registered surface. This milestone does not alter TF10's existing graph or the
original 1–17 development compatibility gate.
