# Publication declaration client contract

`schema.json` and `upstream-profile.txt` are byte-for-byte copies from Platform's
committed machine API candidate identified by `profile.json`. This is separate
from bucket `mock-v5`, the full provisional provider inventory and the peer profile.
The profile is archived as text because its relative links belong to Platform;
use the [original rendered profile](https://github.com/spoke-sh/platform/blob/6627bed1179b7cb95f51831d26d35e207c65481f/contracts/http/publication-profile.md)
to follow the owning schema, vectors and implementation plan.

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
