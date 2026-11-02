# 18. Static application and asset publication

[Roadmap](README.md) · **Status:** Planned

Declare applications, publication targets/configuration and domains through a
Platform-owned API, with import, drift and recoverable lifecycle. Resource names
and schemas remain undecided; the current provider exposes none of this surface.

## Prerequisites and delivery

Consume the HTTP/recovery boundary in [milestone 10](10-live-platform-qualification.md),
scope/access in [milestone 4](04-scope-and-access.md) and real object storage from
[milestone 6](06-databases-and-object-storage.md). These require Platform's actual
P4/P5/P6 contracts; completed provider mocks cannot satisfy them. Platform P7 owns
the [proposed static-publication contract](https://github.com/spoke-sh/platform/blob/ebf97fe086a99ba91f6b669b9f922f3964eb5bfa/contracts/static-publication-v1.md)
and [tracker #29](https://github.com/spoke-sh/platform/issues/29).

1. Agree application/target identities, field ownership, audience, quotas,
   immutable bundle references and expected-revision fences. Decide explicitly
   whether selected heads/promotions are declarative or separately owned runtime
   operations. Domain integration consumes only the qualified static HTTPS slice;
   it need not wait for milestone 15's private connectivity.
2. Pin the public schema and adapt client, fixtures, Terraform and documentation
   together. Keep build/upload/sealing and per-file bytes outside Terraform state.
   Prebuilt delivery does not depend on hosted Nix builds or general containers.
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
