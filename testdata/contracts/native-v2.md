# Native management v2 application boundary

This unregistered codec is superseded by the
[Platform HTTP architecture](../../docs/roadmap/milestones-10-17-implementation-plan.md#architecture-decision-and-scope).
It remains in source until the native-only live/release consumers are replaced;
it is not a planned provider transport or a native SDK adoption requirement.

The byte vectors, capability inventory and embedded observation schema are copied
from Platform `7af27c2bb8083ab64348592f82247da1ff703596`. Their owner is the
[published contract](https://github.com/spoke-sh/platform/blob/7af27c2bb8083ab64348592f82247da1ff703596/contracts/management-native-v2.md).
The native runtime remains pinned to Transit `0e2fba4`; latest Transit's different
ABI definition cannot be substituted without a coordinated platform cutover.

`internal/nativecontract` encodes v2 intent and read context, decodes terminal and
pending results, validates JSON against the platform schema, binds observations
to caller/scope/selectors, and traverses bounded complete snapshots. Logical
fingerprints exclude expected generation; transport identity includes the exact
original signed work. Failed creates may have revision zero and are never success.
V1 codecs and vectors are removed. Duplicate JSON members, old profiles, malformed
receipts and incomplete pages fail without yielding a partial successful result.

These are unregistered application codecs. No local signer, SDK shim or native
HTTP transport is provided. The existing live gate still reports historical
Serving/proof/TLS blockers. Replacing those checks and qualifying the Platform
HTTP contract and real services are pending tasks in the revised plan.
