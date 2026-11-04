# Provisional CI governor contract

`schema.json`, `vectors.json` and `upstream-profile.md` are byte-identical copies
of Platform's `contracts/http/ci-governor-v1.schema.json`,
`ci-governor-v1.vectors.json` and `ci-governor-profile.md` at the source revision
in `profile.json`. The manifest records all three SHA-256 digests. Platform owns
these contract files; this repository retains its MIT license. No upstream
license file is supplied for this separately authored contract, so no existing
provider-schema license is incorrectly attributed to it.

This profile is provisional and has no implemented Platform endpoint. Synthetic
client conformance does not qualify an installed Terraform resource, native
management, GitHub routing, workers, caches, billing or a provider release.
It is independent of mock-v5, bucket metadata and publication-v1. Updates are
explicit hard cutovers, never a negotiation fallback to another profile.
