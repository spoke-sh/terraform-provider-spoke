# Installation, candidate artifacts and releases

The qualified distribution profile is Terraform 1.15.3 / protocol 6 / Linux
amd64 against mock-v5 (see the [current evidence](../qualification/milestones-10-17.md)). No other CLI/system/live API combination is advertised.
Development shells evaluate on four systems; package execution and recovery are
separate evidence. Platform HTTP integration and real service qualification remain open.

## Nix installation

```sh
nix build .#terraform-provider-spoke
nix build .#terraform-provider-spoke-mirror
nix flake check
nix develop --command just test-acc installed
```

The provider filename includes its immutable version. `-version` prints JSON
with version, commit, HTTP contract/version/status/schema digest, Go and system
identity. The embedded [contract identity](../internal/platformclient/contract.json)
is shared with release validation; it currently marks mock-v5 provisional. `VERSION` is the release
series; clean source packages use `VERSION-dev.gFULL_COMMIT`. Dirty snapshots get
an explicitly dirty source-path identity and cannot become release candidates.
The verified Go vendor hash lives in `nix/provider.nix`. The package includes
only approved source/contract inputs, never local state or runtime credentials.

The [independent consumer](../examples/installed-consumer/README.md) pins an exact
committed artifact, nixpkgs, Terraform and lockfile. Its project-scoped CLI config
uses the mirror only for Spoke and allows other Registry providers. Checksums are
enforced with readonly initialization. It requires no provider checkout, compiler
or development override. The test-only fixture is a separate package; starting it
and running Terraform are explicit runtime actions outside Nix evaluation/builds.

Changing the HTTP client, dependencies or package layout requires repeating
package, installed graph and recovery qualification. The provider remains a Go
HTTP client; Transit libraries and ABI alignment belong to Platform. Release
portability still requires executing the actual artifact on clean target hosts.

## Release candidate construction

On a clean committed source checkout, build the mirror and stage it once:

```sh
mirror="$(nix build --no-link --print-out-paths .#terraform-provider-spoke-mirror)"
nix develop --command python3 scripts/release.py build --mirror "$mirror" --output /tmp/spoke-release-candidate
nix develop --command python3 scripts/release.py verify --mirror "$mirror" --output /tmp/spoke-release-candidate
```

The output directory must be new. The builder rejects dirty source, differing
commit identity and unsupported system layouts. It copies the exact executable
into a deterministic ZIP, preserving the same complete payload as the filesystem
mirror. License, source/input hashes and provenance are separate checksummed
assets. The Registry manifest declares protocol 6. It never builds a second
binary for the same version. The verifier rejects missing/extra assets, changed
checksums, truncated archives, mismatched payloads and provenance. Unit checks
also cover immutable publication retries. Actual archive execution uses the
installed Terraform harness, not just `-version`.

Asset names and binary detached GPG signatures follow the official
[Registry publication requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing).
The package builder is the checked-in script running under pinned Python/Nix;
there is no floating GoReleaser download or parallel release compiler.

## Publication gates

The release workflow builds/tests candidates read-only on tag pushes or manual
runs. Manual publication requires selecting the exact `vVERSION-dev.gFULL_COMMIT`
tag and `publish=true`. It then requires full real service-graph evidence for
that artifact from the isolated live runner. Missing/skipped/partial cases and
wrong artifact or dependency identities fail. **This gate currently fails:**
the current HTTP schema is provisional and no real service evidence is qualified.
The validator requires a published Platform HTTP contract pinned in source and
matching artifact/report identities, including schema digest and Platform revision.
It also requires all cases for every advertised family. A fabricated passing
report or relabelled artifact cannot promote the provisional source contract.
Native-profile reports/configuration are rejected. See the
[HTTP boundary record](../qualification/platform-http.md) and
[remaining release plan](roadmap/milestones-10-17-implementation-plan.md#milestone-11-current-profile-qualification-and-publication).

Only the publication job has release-write access and the `provider-release`
environment's `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE` secrets and full
`SIGNING_FINGERPRINT` variable. Keys are imported into a private temporary keyring
outside Nix and removed on exit. A registered RSA/DSA public key and namespace
access must be established with the Registry before any real publication. The
repository is public; namespace/key registration has not been verified.

The publisher compares existing release assets byte-for-byte before resuming,
reuses a valid existing checksum signature, and never uses overwrite/clobber or
deletes assets. It finalizes the draft only after all expected assets are present.
A different artifact always needs a new version, including before 1.0.

No release was published by this change. A passing workflow/candidate/mirror is
not public installation evidence. Milestone 11 still requires anonymous Registry
installation, signature/checksum/readonly verification, the complete installed
live journey, interrupted-download failures and clean-host portability. The
Go HTTP artifact must run without Transit libraries, source, compilers or Nix.
Test actual executable dependencies and TLS trust handling. Do not publish a ZIP
that silently depends on the build machine's store.

Until all roadmap milestones are complete, use hard cutovers between development
contracts. Selecting an older immutable artifact can inspect its own outstanding
work; changing the binary does not undo accepted operations. Never erase state,
replay uncertain work as new intent or force-delete retained data to cross a gate.
