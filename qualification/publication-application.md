# Installed publication application qualification

This record qualifies application declarations through actual Terraform and the
normal installed Platform API. It does not qualify bindings/targets, upload,
sealing, hosted bytes, domains or a Registry release. TF18 remains In Progress.

## Exact inputs

- Provider source/artifact: `581cf627a777c26a27692aaa88508430669e9901`, clean.
- Version: `0.1.0-dev.g581cf627a777c26a27692aaa88508430669e9901`.
- Linux amd64 executable SHA-256:
  `ddf91c61f1087357707eb32ac82d9c06c8c5fd60365b95bb6a044bd0ae93acb3`.
- Platform fixture: `65f8284d0197ccc0ed450dc8c3e0b074947979cb`, clean detached copy.
- Terraform 1.15.3; Go 1.26.7; Linux amd64. Other systems were not exercised.
- Profile: `spoke.platform.http.publication.v1`; schema SHA-256
  `db04fc4ce0e14534c2e44cef932b197e6d9aeb48006f21f7a1523c21def0511f`.
  Copied schema/profile remain pinned to their original Platform source
  `6627bed1179b7cb95f51831d26d35e207c65481f`.

[Machine-readable evidence](publication-application.json) records all native/test
binary hashes and 15 passing case groups, checked at 2026-11-03 01:15:42 UTC.
The run took 239.55 seconds including isolated fixture setup. The separate
[identity protocol proof](publication-identity.json) retains its earlier source.

## Execution and result

```sh
nix build --no-update-lock-file --no-link --print-out-paths \
  .#terraform-provider-spoke-mirror
nix develop --command python3 scripts/qualify-publication-terraform.py \
  --platform /absolute/clean/platform-checkout \
  --mirror /absolute/exact-nix-mirror \
  --report /absolute/new/application-report.json
```

The installed artifact performed validate/apply/refresh/import, empty plans,
A → B → A policy changes, external drift, stale saved-plan refusal, denied refresh
without state loss, protected deletion, recreation, and replacement with distinct
management keys in both replacement orders. Each consumer ran outside the source
checkout, initialized with checksum locks, then used readonly initialization.
Terraform's PATH contained no compiler; missing artifacts fail without rebuilding.

Nine schedules killed the separate provider process during create/update/delete:
before admission, after acknowledged durable acceptance, and after settlement
before Terraform saved the result. A test transport proxy forwards to the real
normal API; it fabricates no receipt or service state. Each schedule separately
restarted Platform's retained native node, owners, courier and gateway through
Platform's fixture control before retry. Original receipts and one resource
incarnation survived; operation counts exclude duplicate admission. Each group
used a fresh isolated installation to respect lifetime capacity limits.

Initial candidate `a60da488` passed ordinary installed lifecycle but failed two
immediate create retries: a temporary rate limit and still-settling accepted
receipt. The final artifact adds bounded GET retries and observation of the
original receipt, preserving keys and revision fences. All schedules then passed.
Interrupted HTTP connections during deliberate service kills are expected fixture
logs, not passing substitutes for a failed case.

Focused race tests and the synthetic application lifecycle/nine crash schedules
also passed. `just fmt`, `just check` and `just build` passed in the locked Nix shell. The
full mock acceptance run passed in 1,136.461 seconds; the focused recovery/client
checks also passed after the native-run fixes.
synthetic results remain separate from this real local API evidence. Latest
upstream CI is tracked in the [adoption plan](../docs/roadmap/platform-adoption-implementation-plan.md),
including the failed upstream publication-api fixture assertion. Local success
here does not relabel that upstream run.

## Limits

The API remains an operator-provisioned finite development profile: eight lifetime
application identities and 64 lifetime receipts per owner, including deleted or
expired records. A new stable management key is required for single-apply
replacement; same-key forced replacement fails closed rather than advancing a
saved mutation fence. Imports require retained ownership evidence. Unknown or
expired outcomes cannot authorize fresh keys or state deletion.

The current independent consumer still pins its earlier bucket artifact; updating
the complete catalog/nightly/production consumer is the next review unit after
bindings/targets. Local state files do not qualify a hosted encrypted/locked
backend. No real object service, website bytes, domain or cross-platform runtime
claim follows from this record.
