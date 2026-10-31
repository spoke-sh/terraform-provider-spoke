# Milestones 2–4: mock contract, recovery, scope and access

Recorded 2026-10-31. This is provider/mock qualification, not a live platform
service or production durability claim.

## Source and environment

The implementation tree is committed as
[`32d6894`](https://github.com/spoke-sh/terraform-provider-spoke/commit/32d6894655b6d1ad8d52e1f51f849e2702ae6a08),
following contract/discovery commit `050fe05`, recovery commit `a3eafa1` and
scope/access commit `dbcf72b`. Commit `32d6894` completes exact-target authorization.
Execution used the locked Nix shell, Go 1.26.7, Terraform 1.15.3 and Linux amd64.
The platform review used `3f155dd21c6f6dd4aaa7a44e52ffc60aec3b0740`; the
[capability matrix](../testdata/contracts/capabilities.md) records missing public
APIs and their upstream owners. No platform repository or live service was changed.

## Executed checks

From the locked Nix shell, `just fmt`, `just check` and `just build` passed.
Check/build were rerun after the final exact-target authorization fix:

- Nix/Just/Terraform/Go formatting, actionlint, host flake checks and Go vet.
- Unit, raw HTTP/schema, client and fixture tests with the race detector.
- All **14 top-level mock Terraform acceptance tests**, including their
  per-resource and interruption subtests, with the race detector enabled for
  the test harness. Ordinary unit tests leave `TF_ACC` unset; the separate
  required acceptance invocation sets `TF_ACC=1` and `SPOKE_TEST_TARGET=mock`.
- Provider executable build. Generated schema docs were refreshed with
  `tfplugindocs generate --provider-name spoke`.

Installed-consumer tests compile a normal provider binary, create a private
versioned filesystem mirror, verify a checksum lock and readonly re-init, then
run Terraform with no Go in its PATH and no development overrides. Each provider
configuration/alias runs in its actual Terraform-managed process. In-process
reattachment cannot model independent alias configuration safely and is not used
for that case. The test-only `0.0.1` mirror is not a published release artifact.

## Evidence by milestone

| Milestone | Executable owner | Outcome |
| --- | --- | --- |
| 2: contract and discovery | [Raw HTTP/schema tests](../internal/testutil/mockplatform/server_test.go) and [client discovery tests](../internal/platformclient/discovery_test.go) | Version/authentication/target gates, concrete wire fixtures, complete pagination, stale/forged/repeated tokens, partial/denied pages, snapshot/identity checks, bounds, cancellation, retry hints, quotas, distinct tenants/principals and expired receipt tombstones. |
| 3: recovery and isolation | [Client recovery tests](../internal/platformclient/recovery_test.go), [lifecycle tests](../internal/platformclient/client_test.go) and [installed process tests](../internal/provider/process_linux_test.go) | Original-operation recovery, independent-state ownership, terminal failures, retained unrelated resources, pending timeouts and state preservation. |
| 4: scope and access | [Terraform scope lifecycle tests](../internal/provider/scope_resource_test.go), [scope/access client tests](../internal/platformclient/scope_test.go) and [installed process tests](../internal/provider/process_linux_test.go) | Projects, environments, service accounts and grants; import, empty plans, rename/drift, protected deletion, authoritative absence, interrupted CRUD, denial, explicit grants (including exact-target management without sibling disclosure) and dependency-restricted teardown. |

The installed crash matrix kills the provider at **12 deterministic points**:
create/update/delete crossed with before mutation dispatch, before fixture
acceptance, after persisted acceptance, and after completion before its reply.
Each case restarts the fixture service and reloads its ledger, completes pending
work explicitly, retries through Terraform and checks operation/resource counts
and original identities. No arbitrary sleep establishes admission or completion.

Additional installed tests prove:

- A pending operation times out with recovery context, then resumes without a
  second create after fixture restart. A failed read preserves the state file.
- Two Terraform states with different principals cannot replay or import the
  other's ownership; same-name objects with distinct keys remain independent.
- Organization-scoped bootstrap supplies concrete project/environment IDs to a
  separate bucket consumer. Parent teardown refuses to cascade into that bucket.
- Removing a self-owned final grant can finish server-side while its receipt
  becomes unreadable. Terraform retains state until an independent grant restores
  inspection authority; recovery does not delete that independent grant.
- Saved plans for all four new resource kinds reject an external revision change
  without overwriting it. An explicit refreshed plan then converges on the same ID.
- Organization, project and environment provider aliases read all five collection
  types, including multiple pages. A denied collection read preserves known state.
- Synthetic credentials are absent from Terraform state, saved-plan JSON/raw
  files, trace logs and diagnostics, including a secret-bearing failure response.

The raw fixture and client do not call each other's retry/lifecycle implementation.
Schema/golden checks and server ledger assertions supplement Terraform state
assertions. Bindings and their additional deletion rules remain milestone 7.

## Limits and remaining gates

Fixture restart closes/restarts its HTTP service and deserializes the private
synthetic ledger in the test harness. The provider is a separate killed process;
the fixture is not a separately killed production server or a power-failure test.
This does not establish real controller, cloud or storage durability.

Identical principal credentials plus management key remain indistinguishable
from legitimate recovery. Ownership transfer, unmanaged adoption and automatic
reuse of expired keys are unsupported. A matching terminal failure remains the
original result for identical intent; it is not silently retried as a new operation.
These explicit draft constraints require platform agreement before live adoption.

There is no OCI/object/database data plane in the fixture. Additional service
families, Nix release packages, other Terraform versions/systems, platform-owned
contracts and real endpoints remain milestones 5 onward. Live mode still fails
explicitly. CI's multi-system flake evaluation does not qualify execution on
those systems; only the Linux amd64 profile above has new execution evidence.
