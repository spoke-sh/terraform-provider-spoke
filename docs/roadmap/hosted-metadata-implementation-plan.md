# Implementation plan: hosted metadata consumer qualification

[Roadmap](README.md) · [Milestone 10](10-live-platform-qualification.md) · **Status:** In Progress

Exercise an exact installed provider against an explicitly supplied Platform
HTTPS API using only consumer authority. This repository implements Terraform
integration and consumer acceptance. Platform deployment, native activation,
operator controls and backend qualification belong upstream.

Implementation is authorized. The user clarified on 2026-11-05 that this work
must not deploy or administer Platform. No hosted target is provisioned yet.
Complete the consumer implementation and its synthetic/installed checks, and
record hosted execution as blocked until an isolated service is supplied.

## Scope and evidence

The [review](platform-adoption-implementation-plan.md#hosted-metadata-planning-review-2026-11-05-0039-utc)
records provider `ac5cbf4`, Platform `3d397d1` and the public website with its
runtime disabled. The website is not a management endpoint. The existing
`spoke.platform.http.bucket.v1` development profile provides bucket declarations
and original-operation recovery, not usable storage.

Use the existing schemas and Go HTTP client. No new resource, Transit dependency,
cloud SDK, provisioning script, native credential or administration route is
needed. Leave independent consumer pins and release requirements unchanged.
Terraform readiness remains pending with no storage endpoint.

The resulting evidence must identify itself as consumer metadata qualification.
It does not establish the deployed server's source/artifact identity, retained
server restart, receipt expiry, backend durability, usable storage or release
readiness. Operator-supplied Platform revision metadata is attributed as supplied,
not independently attested by the current capabilities API.

## Delivery sequence

### 1. Explicit selection and report boundary

Add `gate: hosted-metadata` and
`evidence_kind: hosted-platform-metadata-consumer` to the existing live task.
Use a separate case inventory; do not add cases to the full-service manifest
whose entries the release script aggregates.

Require an explicit HTTPS origin, scope, expected authenticated principal,
external CA/token file paths, exact provider mirror/version/hash, clean harness
revision, run ID, private run directory and bounded deadlines. A second external
credential may exercise foreign-owner denial where supplied by the service;
the required baseline uses an intentionally invalid credential for authentication
denial and never creates service accounts or changes server grants.

Validate the configuration and installed bytes before HTTP, then verify
TLS/capabilities/profile/principal/scope before mutation. Reject mock profiles,
S3/HTML responses, redirects, mismatched identity and unsupported inputs.
Keep local qualification separate. No ambient credentials or source-build
fallback may enable the hosted mode.

Bind reports to the exact artifact, harness, Terraform/system, endpoint/scope,
run and observed HTTP profile. Preserve explicit upstream evidence limitations.
Synthetic tests and local fixture reports cannot validate as hosted reports;
consumer metadata reports cannot pass the full service/release gate.

### 2. Installed consumer lifecycle

Run an external Terraform root with the selected mirror, exact version and
readonly checksums. Terraform processes have no compiler or ambient CLI/plugin
configuration. Reuse the existing client and installation patterns.

Persist each run's inputs and known management keys before requests. Reopening
an interrupted run must preserve its original keys/state; refuse incompatible
inputs. Never generate a successor because a previous outcome is unknown.
Use only public HTTP operations for setup, drift and cleanup. No Platform
checkout, AWS access, operator script or native restart command is required.

Required lifecycle coverage: create/read/import, exact/collection discovery,
empty plans, rename/update and A → B → A, external drift, revision conflicts,
invalid credential state preservation, protected deletion and authoritative
absence. Check that declarations still report unavailable data-plane readiness.

### 3. Consumer interruption and observational behavior

Use an isolated test relay only for deterministic transport interruption.
The relay verifies upstream TLS, restricts forwarding to the selected origin
and preserves request bytes. Direct-origin lifecycle results remain separate
from relay results.

For create/update/delete, exercise three boundaries:

- Before forwarding: prove the client request was not dispatched.
- After receiving acceptance but before Terraform saves it: recover the original
  receipt and resource. The server may already have settled; do not claim the
  operation was held pending.
- After independently observing terminal success but before state save: recover
  the completed original operation, including deletion.

Kill the actual provider process and retain the consumer root. Assert original
operation identity and no additional logical mutation on retry. Bound barriers
and polling; reads/refresh/plan must not dispatch writes. Do not pause a courier,
restart a service, expire receipts, revoke a principal or inspect private state.

Server restart, settlement-time revocation, token issuance/rotation, controlled
quota exhaustion and receipt expiry remain Platform-owned qualification. Link
their separate evidence when available; do not mark those scenarios passed by
consumer tests or include them as silently skipped consumer cases.

### 4. Bounded cleanup and confidential evidence

Cleanup only known resources/keys created by this run, through the normal API
and original-operation recovery. Refuse destructive cleanup of uncertain,
protected or differently owned work. Retain failed state for explicit recovery.
A failing run cannot reset the scope or relabel uncertainty as absence.

Write private, fresh, checkpointed reports. Require every selected case and its
assertions to pass. Deadlines terminate owned local processes while preserving
remote operation identities. Check credentials do not appear in Terraform state,
plans, diagnostics or publishable reports. Keep raw private evidence outside
Actions artifacts and publish only a sanitized summary.

### 5. Workflow, documentation and PR

Extend the existing manual live workflow and five-task interface. Ordinary CI
remains synthetic and credential-free. The same explicit local command can run
against a hosted target when supplied:

```sh
SPOKE_LIVE_CONFIG=/private/consumer-metadata.json \
SPOKE_LIVE_REPORT=/private/consumer-metadata-report.json \
  nix develop --command just test-acc live
```

Update README, HTTP guidance, milestone 10 and this plan with implemented behavior
and executed evidence. Submit one provider PR with logical commits for selection/
reports, installed lifecycle, interruption/recovery, and workflow/documentation.
There is no Platform deployment or companion implementation PR in this slice.

## Validation and completion

Run locked-shell `just fmt`, `just check`, `just build` and
`just test-acc installed`. Add focused configuration, report freshness, identity,
release rejection, process/deadline, transport and credential tests. The actual
installed consumer must execute against an explicitly synthetic HTTPS fixture in
ordinary/installed acceptance; that evidence cannot become hosted qualification.
Exercise affected local metadata/declaration dispatch regressions. Regenerate
reference docs if examples or schemas change; evaluate all systems if Nix changes.

Code completion requires the implemented runner and passing synthetic/installed
checks. Hosted consumer qualification additionally requires an externally
provisioned isolated API, current consumer credentials, expected scope/principal
and a complete fresh execution. Missing service inputs produce a clear blocked
result. TF10/TF11 remain In Progress, and backend/service/release gates remain
separate after a consumer pass.
