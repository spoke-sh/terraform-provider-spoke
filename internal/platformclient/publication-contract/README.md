# Publication declaration client contract

`schema.json` and `upstream-profile.md` are byte-for-byte copies from Platform's
committed machine API candidate identified by `profile.json`. This is separate
from bucket `mock-v5`, the full provisional provider inventory and the peer profile.

The Go client validates requests and responses with the owned schema, including
required/null fields, exact decimal counters and bounded complete observations.
External schema loading is disabled. Server fingerprints stay opaque: an exact
HTTP replay validates the original intent and principal against retained state.
No native command encoding, SDK, signer or runtime belongs in this provider.

The client supports application, binding and target policy plus observational
receipt/history access. It has no upload, promotion, billing or cloud API. This
commit does not register Terraform resources or qualify hosted static delivery.
The remaining lifecycle adapter and installed/static qualification gates belong
to [milestone 18](../../../docs/roadmap/18-static-publication.md) and consumer
[#13](https://github.com/spoke-sh/terraform-provider-spoke/issues/13).
