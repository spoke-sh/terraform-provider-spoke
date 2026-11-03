# Independent installed consumer

This Linux amd64 example pins provider commit `bc1860930924ee5c0a88004273eb85bedf54cfe2`,
Terraform 1.15.3 and HashiCorp Random 3.7.2. Its flake and Terraform lockfiles select
immutable inputs. Spoke installs from the Nix mirror; Random uses its ordinary
Registry installation and signed checksums. No provider source, development
override, compiler, credential or Terraform state belongs in the consumer flake.

Copy this whole directory to another workspace, then run:

```sh
nix develop --command bash ./check.sh
```

The shell has only Terraform, the separately packaged synthetic fixture and small
shell utilities on PATH. Shell entry and Nix builds do not start the fixture or
apply Terraform. The explicit check starts a local TLS fixture in a new private
temporary directory, supplies runtime credential/CA file paths, runs readonly
initialization, CRUD/import/empty-plan checks, and destroys its synthetic bucket.
It prints the retained fixture evidence directory and stops the process on exit.
It creates no real platform services or object bytes. Do not use this fixture as
a deployment server. Run from a fresh directory; existing state is never erased
to accommodate a contract change.

The root `just test-acc installed` additionally exercises the full graph, aliases,
retained data canaries, lost replies and killed-provider recovery against the exact
Nix mirror artifact. The Go harness controls the synthetic server outside the
consumer; Terraform's PATH contains no compiler. Provider process discovery
verifies the actual executable through Linux `/proc` before killing it.

To deliberately update the pin, change the provider revision and exact constraint
together. Build the trusted mirror, generate its lock with
`terraform providers lock -fs-mirror=PATH -platform=linux_amd64 spoke-sh/spoke`, then lock
Random from its origin and review both lockfiles. Normal runs use
`-lockfile=readonly`; never repair a checksum mismatch by disabling verification.
Mirror hashes prove local artifact selection, not a signed Spoke Registry release.
Spoke has no public Registry release yet.

## Independent publication declarations

The same exact mirror also provides a three-root declaration example:
`publication/catalog` owns the application and two reservations;
`publication/nightly` and `publication/production` own their separate targets.
The target roots receive reservation IDs from catalog outputs, use separate
credential files and keep independent state. They manage policy only; there are
no upload, runtime-selection, domain or hosted-state claims.

The explicit qualification harness copies this complete consumer outside the
provider checkout, verifies the pinned mirror bytes, and executes
`check-publication.sh /absolute/fixture.json` in this compiler-free Nix shell.
The configuration supplies `endpoint`, `ca_file` and `tokens` paths for `project`,
`nightly` and `production`; operator bootstrap stays in Platform. Normal runs
initialize all three roots with their committed readonly checksum locks and
check import, empty plans, independent policy updates, protection and teardown.
The example is for fresh isolated fixtures: it must not reuse customer state.
A deliberate checksum update generates each root's lock from the same trusted
mirror. Do not copy tokens or state into this directory or the Nix store.
