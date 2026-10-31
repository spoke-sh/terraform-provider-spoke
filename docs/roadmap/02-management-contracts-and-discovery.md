# 2. Management contracts and bounded discovery

[Roadmap](README.md) · **Status:** Completed

Give every resource a versioned management contract and let callers distinguish
supported capabilities, authorized absence and incomplete discovery.

## Prerequisites and scope

Build on [milestone 1](01-mock-provider-foundation.md). Platform owns the real
contract; this milestone owns the provider capability matrix, draft fixture
coverage and adoption path. Review platform source now and record exact versions
and upstream gaps. Mock development may continue while publication is pending;
[milestone 10](10-live-platform-qualification.md) owns completed live adoption.

## Delivered implementation and evidence

The [mock-v2 contract](../../testdata/contracts/README.md) adds explicit hierarchical
targets, authenticated capabilities, bounded complete discovery, principal-bound
receipts and durable expiration tombstones. The historical mock-v1 specification
is preserved. The [capability matrix](../../testdata/contracts/capabilities.md)
records every initial resource's operations, scope, recovery/deletion requirements
and missing platform API owner at platform revision `3f155dd`.

[Executed qualification](../../qualification/milestones-2-4.md) covers raw HTTP and
schema fixtures, client boundary failures, pagination, cancellation/retry bounds,
quotas, independent owners/tenants and receipt expiration. The implementation
tree is `32d6894`. Transfer and credential-cloned state isolation are explicitly
unsupported in this draft. Published platform agreement and live adoption remain
milestone 10; completion here qualifies the provisional development contract.

## Delivery sequence

1. Record a capability matrix for every initial resource: owner, versioned
   operation, authentication, target scope, identity, CRUD/import, preconditions,
   recovery, dependencies, deletion semantics and evidence or missing contract.
   Link verified platform owners instead of inferring APIs from UI surfaces.
2. Specify authenticated target/capability discovery and unsupported-version
   behavior. Keep provider configuration free of network-dependent provisioning;
   validate negotiated target and required capabilities before mutation.
3. Define opaque paginated discovery with stable selection rules, total page/item
   bounds, cancellation, repeated-token detection and explicit partial results.
   Add project/environment and relevant service reads without treating truncated
   lists or masked 404s as authoritative absence.
4. Resolve management ownership, fingerprint encoding, operation generations,
   receipt retention/expiry, pending discovery, import adoption and transfer.
   State what external identity distinguishes independent owners. Identical
   credentials and keys are currently indistinguishable from legitimate recovery;
   do not promise cloned-state protection without an enforceable contract.
5. Extend the draft schema and independent raw-HTTP tests before client changes.
   Specify quota, dependency, rate-limit and unavailable errors, rejected
   operations, retry hints and bounded settlement. Use injected timing controls.
6. Record draft-to-platform divergences. When a published schema is available,
   pin it and derive boundary types; otherwise keep the draft label and each
   unresolved upstream prerequisite explicit. Do not silently reinterpret mock-v1.

## Acceptance criteria

- Every initial resource has a contract/capability row, including missing API
  work with an owner and a blocking gate for live qualification.
- Raw HTTP and actual-client tests cover capability/version mismatch, wrong
  target, all discovery pages, repeated tokens, partial/denied pages, total bounds
  and cancellation. Reads submit no mutations and disclose no other tenant.
- Ownership, receipt expiration and conflicting replay rules have executable
  positive and negative fixtures. Any indistinguishable cloned-state case is
  explicitly unsupported rather than described as isolated.
- Unsupported routes and masked denials cannot erase state. Quota and dependency
  errors remain distinguishable from successful completion or absence.
- The published contract or explicitly provisional successor is documented with
  source revision and outstanding platform decisions. Live support remains gated
  by milestone 10 even when this mock development contract is complete.
