# Publication Terraform identity protocol

The adapter uses Platform's existing principal-bound, immutable operation-key
namespace. It does not add a native codec, local journal or new HTTP endpoint.
The [normal publication profile](https://github.com/spoke-sh/platform/blob/f43f70d9defc34b9415bb28a6a3bdfceb147c726/contracts/http/publication-profile.md)
atomically retains each key, full intent and original operation. Matching replay
returns that operation; different intent conflicts; expiry never frees its key.

## Retained slots and mutation fences

A management key contains 1–24 identifier characters. Operation keys are
`tf1a_<management_key>_<NN>` for applications, with `b`/`t` reserved for bindings
and targets, and `NN` a two-digit decimal slot from `00` through `63`. Each key
fits the API's 32-byte bound. Resource kinds and exact project/environment scopes
have independent namespaces. This format is a provider protocol over public
receipts; fingerprints remain opaque Platform commitments.

Read occupied slots from zero until the first authoritative 404. Every receipt
must belong to the configured principal and expected resource kind/incarnation.
Denied, malformed, unavailable, expired or unsettled results stop discovery.
Reads never submit or advance work. Permanent slots make a complete prefix
observable without merging mutable collection snapshots.

The next slot is a compare-and-set on logical intent: concurrent callers choose
the same key. Atomic server admission accepts at most one full intent there.
Identical intent replays; different intent conflicts. A mutation uses the slot
selected by its saved plan/prior state, plus its original resource revision.
It never advances that fence during apply because a newer read happened to pass.
Resource revision conflicts remain independent of this mutation sequence.

This construction requires clients using the same management identity to follow
the contiguous-slot protocol. The service's immutable keys enforce the conflict;
the service does not interpret `tf1` prefixes or expose a Terraform generation
field. Manual writes into this reserved namespace can block recovery and must
not be used to repair or skip an operation. Authorized non-Terraform policy
updates remain external drift; they do not transfer management ownership.

## Recovery and ownership

After death before Terraform saves an ID, configuration selects the same prefix.
The retained create receipt supplies the original resource ID. A fresh plan can
recover that owned declaration. If its policy differs from the new configuration,
restore the accepted configuration or import the exact resource before planning
new intent. Never adopt an unrelated object by name.

An update A → B → A consumes two new slots. Delete retains its slot and old
incarnation; recreation uses a later create slot and a new server resource ID.
An old saved plan still addresses its original slot/revision and cannot be
silently rewritten to affect a replacement. Expired receipts are an explicit
recovery error, including when an application still exists. There is no automatic
reclamation, import workaround or erasure of retained services.

Import requires an authorized complete history and a retained create receipt
whose principal, resource ID and supported management prefix match. It then
rechecks the complete prefix and exact resource. Read permission alone cannot
claim an unmanaged declaration or another principal's namespace. Same-principal
credential rotation preserves ownership. Identical credentials/keys cannot
distinguish a cloned Terraform state from recovery; use one state owner and
external state locking.

Use a new management key for overlapping replacement. A create using an existing
live identity recovers that incarnation; it cannot allocate a second live object
under the same identity. After authoritative deletion, the same key can recreate
in a later slot. Scope changes must not silently rebind an existing canonical ID.

## Bounds and evidence

There are at most 64 receipt identities per Platform owner across all callers
and resources, not 64 available mutations per Terraform resource. Deleted
application IDs and expired receipts still consume installation capacity. This
is a finite development profile; exhaustion fails rather than pruning history.

`TestPublicationIdentity*` exercises occupied-slot conflicts, stale fences,
A → B → A, recreation, principal mismatch, incomplete/expired evidence and key
bounds. `TestPublicationPlatformIdentityProtocol` executes the construction
through the installed normal Platform API. Its first clean result at provider
`8e225b6289977dc655575e4974460e3ac8f39d93` and Platform
`f43f70d9defc34b9415bb28a6a3bdfceb147c726` passed in 6.473 seconds including
fixture setup, with five reported case groups. This is protocol evidence; exact
installed Terraform and process-death results are recorded separately.
