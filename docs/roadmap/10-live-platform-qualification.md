# 10. Live platform qualification

[Roadmap](README.md) · **Status:** Planned

Prove that the initial Terraform service graph controls real platform services,
recovers original accepted work and leaves retained data usable after compute teardown.

## Prerequisites and scope

Contract coordination starts in [milestone 2](02-management-contracts-and-discovery.md).
Complete this milestone using the [recovery matrix](03-recovery-isolation-and-protection.md),
initial graph in [milestones 4](04-scope-and-access.md)–[8](08-container-services-and-endpoints.md)
and [installed artifact](09-nix-packages-and-consumers.md). Platform must provide
published versioned management APIs, automation auth, service controllers and
an isolated qualification environment. Provider mocks cannot satisfy those
upstream prerequisites. Qualify resources incrementally as their APIs become available.

## Delivery sequence

1. Pin the platform API/schema and exact platform revision. Reconcile scope,
   automation credentials, IDs, errors, capability negotiation, revisions,
   acceptance/settlement, ownership, receipt retention and deletion semantics.
   Update boundary types, fixture contracts, client, Terraform schemas and docs
   together; preserve evidence of each divergence from the provisional protocol.
2. Implement an explicitly selected live lane through `just test-acc live` and
   a separate configured workflow. Require endpoint/CA/tenant/credential and
   isolated fixture configuration; fail if absent or if no tests are selected.
   Never infer live mode from environment credentials or fall back to mocks.
3. Keep backend provisioning/setup in the harness. Use local/isolated Terraform
   state, fixture-owned identities and bounded cleanup. Do not require customer
   Terraform to create the management endpoint used to configure its provider.
4. Execute applicable contract and Terraform lifecycle scenarios on real APIs.
   Kill the provider, lose replies, restart platform/controller components and
   retry accepted CRUD. Verify authoritative backend receipts and resource IDs.
5. Exercise the real data plane independently: registry push/pull of the declared
   digest, bucket upload/download, database read/write, workload access through
   bindings, endpoint reachability/TLS and access revocation. Retire compute and
   read the separately retained data again through authorized clients.
6. Publish a capability matrix with exact provider/platform/API/artifact/CLI/system
   revisions, executed runs, failures and exclusions. Promote only qualified
   resource operations from experimental to supported documentation.

## Acceptance criteria

- The complete initial service graph passes lifecycle, import, empty-plan,
  drift, denied access, revision conflict and dependency-safe teardown against
  the real backend using the installed provider.
- Interrupted operations and server/controller restart preserve original intent
  without duplicate logical resources. Uncertainty remains inspectable; timeout
  or lost authority does not imply rollback or authoritative absence.
- Real OCI, object, database and workload/endpoint assertions pass, including
  negative authorization and retained-data checks after compute removal.
- Ordinary PR CI remains credential-free. Live test selection, isolation and
  cleanup are explicit; skipped/missing cases cannot qualify a capability.
- Executed evidence and remaining upstream limits are recorded. Neither a
  successful management response nor a running process alone establishes a
  service guarantee, broader backend support or public distribution.
