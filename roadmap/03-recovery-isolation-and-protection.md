# 3. Recovery, isolation and data protection

[Roadmap](README.md) · **Status:** In Progress

Prove the reusable lifecycle preserves accepted intent, state and unrelated data
through interruption, concurrent callers and denied or incomplete observations.

## Prerequisites and scope

Build on [milestone 1](01-mock-provider-foundation.md) and the ownership/error
contracts in [milestone 2](02-management-contracts-and-discovery.md). Extend the
first bucket before copying its patterns into the initial service graph. These
are deterministic mock protocol proofs; real service durability belongs to
[milestone 10](10-live-platform-qualification.md).

## Delivered foundation

Existing tests cover lost mutation replies, generations, pending observation,
concurrent identical create, stale revisions, denied reads, protected/nonempty
deletion and a provider-process death after create acceptance. The server stays
alive in that process test; its in-memory ledger is not restart durable.

## Delivery sequence

1. Add deterministic barriers before dispatch, after persisted acceptance,
   before response and after completion but before Terraform records the result.
   Keep progression independent of GET and replace arbitrary timing assumptions
   with explicit controller signals and bounded waits.
2. Persist only synthetic fixture resources, receipts, revisions and generations
   in a test temporary directory. Restart the fixture against that ledger and
   prove replay and retention rules without using provider retry code as an oracle.
3. Extend separate-provider-process acceptance to create, update and delete at
   each relevant fault point, including lost terminal replies and pending work
   past timeout. Restart with the same intent and inspect original receipts.
4. Exercise independent Terraform states with distinct ownership, same-name
   objects, concurrent changes, stale revisions and explicit import/transfer.
   Test the enforceable milestone 2 contract; retain its identical-key/credential
   limitation unless a backend ownership mechanism actually resolves it.
5. Seed multiple tenants and unrelated resources. Revoke permission while work
   is pending and test cross-scope IDs, malformed/oversized replies, quota refusal,
   rate limiting, outage and incomplete pages. Assert bounded attempts and time.
6. Inspect state, saved plans, provider logs and diagnostics for synthetic secret
   canaries. Assert server ledger and resource state before cleanup; preserve
   evidence of original operations and cleanup only known fixture identities.

## Acceptance criteria

- Interrupted CRUD and fixture restart recover the original accepted operation;
  there is exactly one logical resource and no accidental replay of a later
  return to old configuration. Timeout never claims rollback.
- Concurrent incompatible intent conflicts without overwriting the winner or
  taking over another owner. A renamed object retains its ID; a new same-name
  object is neither adopted nor confused with the removed object.
- Denial, unavailable services and incomplete discovery preserve known state.
  Nonempty/protected deletion is refused and unrelated resources survive.
  Binding-specific deletion constraints are added with milestone 7.
- Negative paths prove bounds and redaction, including failed operations and
  pending timeouts. Missing selected acceptance cases cause failure, not a pass.
- CI runs the required mock suite without credentials. The evidence records
  fixture restart separately from provider restart and neither claims production
  backend durability. Every scenario in issue #1's step 5 has an explicit owner
  and executable result before its corresponding guarantee is reused.
