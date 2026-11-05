# Hosted metadata consumer qualification

The `hosted-metadata` gate tests consumers of an already available Platform HTTPS
API. It runs the exact installed provider through Terraform; it does not deploy
Platform, create cloud infrastructure, restart a server or use operator credentials.
**No hosted target has been supplied.** Synthetic execution of this harness is
implementation evidence only. Full service qualification and releases remain blocked.

The [execution record](../qualification/hosted-metadata-consumer.md) binds the
passing synthetic/installed cases to their exact harness and artifact revisions.

Platform must supply an isolated, initially empty environment implementing the
pinned `spoke.platform.http.bucket.v1` profile, a scoped consumer machine token,
its expected principal ID and TLS trust. The public website is not this API.
The profile manages bucket declarations; readiness remains `pending` with reason
`dependency_unavailable`, no active revision and no usable storage endpoint.

## Select an exact consumer run

Use a clean committed provider harness and a Linux amd64 immutable mirror.
Nix may build that selected artifact before the run; Terraform has no compiler,
source-build fallback, development override or registry fallback. Its first init
records the mirror checksum; subsequent initialization uses a readonly lock.
The independent committed consumer's pins and locks are unchanged by this gate.

Create a private configuration outside the checkout and Nix store. Replace every
placeholder below with the selected artifact/target value. The artifact's
`-version` JSON supplies its version, revision and contract digest. The profile
digest is pinned in `internal/platformclient/profile.json`.

```json
{
  "gate": "hosted-metadata",
  "evidence_kind": "hosted-platform-metadata-consumer",
  "api_version": "mock-v5",
  "api_schema_sha256": "<contract_schema_sha256 from the executable>",
  "management_profile": "spoke.platform.http.bucket.v1",
  "profile_schema_sha256": "8568a6483a450b79464386d9cd6e5be5c7ad0c4163ceecd0ffe3a7682b4341bd",
  "harness_checkout": "/absolute/clean/provider-checkout",
  "harness_revision": "<40-character clean harness commit>",
  "provider_revision": "<40-character artifact source commit>",
  "provider_version": "0.1.0-dev.g<artifact source commit>",
  "mirror": "/absolute/installed/mirror",
  "artifact": "/absolute/installed/mirror/registry.terraform.io/spoke-sh/spoke/0.1.0-dev.g<artifact source commit>/linux_amd64/terraform-provider-spoke_v0.1.0-dev.g<artifact source commit>",
  "artifact_sha256": "<SHA-256 of those exact executable bytes>",
  "platform_revision": "<40-character revision supplied by the Platform operator>",
  "endpoint": "https://isolated.example.invalid",
  "ca_file": "/private/consumer/ca.pem",
  "credential_file": "/private/consumer/token",
  "expected_principal": "consumer-principal",
  "scope": {
    "organization_id": "org",
    "project_id": "project",
    "environment_id": "env"
  },
  "run_id": "isolated-consumer-check",
  "run_directory": "/private/consumer/runs/isolated-consumer-check",
  "deadline_seconds": 1800,
  "max_objects": 32
}
```

The run directory's parent must exist. The directory itself must be new or empty
and mode `0700`; the runner creates it if absent. Credentials must be private
regular files, at most 1 MiB, outside the run directory and Nix store. The endpoint
is an HTTPS origin without a trailing slash, path, query or user information.
Scope and principal must already exist. No fixture checkout or administrative
control can be selected. The artifact and clean harness are checked before HTTP;
TLS, bounded profile, authenticated principal and scope are checked before mutation.
The overall deadline is at most 30 minutes. Each Terraform command has a
45-second budget; provider lifecycle operations and relay settlement observation
each have a 30-second budget. A timeout preserves uncertainty and fails the run.

From the selected provider checkout's locked shell:

```sh
SPOKE_LIVE_CONFIG=/private/consumer/config.json \
SPOKE_LIVE_REPORT=/private/consumer/summary.json \
  nix develop --command just test-acc live
```

The existing manual live workflow accepts this same explicit configuration.
It uploads only the summary; never upload the run directory, Terraform state,
plans, raw logs or credential files. Ordinary CI selects the synthetic fixture
explicitly and cannot use these inputs to enter the hosted child driver.

## What executes

Fifteen required cases cover direct CRUD/import, exact and collection discovery,
empty plans, A → B → A revisions, drift, stale saved plans and HTTP revision
conflicts, invalid-credential state preservation, protected deletion and
authoritative absence. A separate relay checks observational reads and nine
provider deaths: create/update/delete before forwarding, after receiving the
acceptance reply, and after an independent read observes terminal success.

The relay uses verified upstream TLS, forwards only within the selected scope,
preserves request bytes and never controls the server. Each crash kills the actual
installed provider, leaves Terraform to report the failure, and retries using
the retained root. Receipt ID, fingerprint, generation, owner and resource identity
must remain consistent; no extra logical mutation may appear. Acceptance may
already be terminal: this test does not claim to hold server work pending.

Cleanup deletes only known, unprotected resources from this run through normal
consumer APIs. It never resets a scope or deletes resources found by listing.
The final scan checks that the external credential is absent from private state,
plans, diagnostics and reports. A test failure preserves evidence and uncertain work.

## Interrupted runs and evidence limits

Keep the same configuration, run ID and directory when resuming. `inputs.json`
binds the run; atomic private journals retain step completion, management keys,
revision fences and original operation observations. Changed inputs and concurrent
runs are refused. Previously completed steps are retained, not repeated as new
mutations; a resumed summary preserves the original start time and identifies the
new attempt. Completed runs cannot be reused as fresh evidence.

If the driver dies while recording the crash itself, it cannot prove whether the
provider was killed at the intended boundary. That case fails closed. Inspect the
retained journal and original public management receipt, then recover through the
same Terraform root and original configuration. Do not erase state, allocate a
replacement key, or rerun an interrupted mutation under a new identity. A separate
qualification run is appropriate only after the original work has been resolved.
The temporary relay origin is local to an attempt; direct recovery uses the original
upstream origin and CA from `inputs.json`, preserving all resource fields and keys.

The report attributes the supplied Platform revision to its operator: the current
API cannot attest deployed server bytes. Server restart, receipt expiry, durable
backend execution, principal revocation, storage bytes and service readiness need
separate Platform evidence. Consumer metadata reports have their own inventory
and cannot pass the service-graph or release gate. See the
[implementation plan](roadmap/hosted-metadata-implementation-plan.md) and
[milestone 10](roadmap/10-live-platform-qualification.md).
