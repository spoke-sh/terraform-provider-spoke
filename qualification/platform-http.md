# Platform HTTP boundary implementation

Implementation follows the revised [milestones 10–17 plan](../docs/roadmap/milestones-10-17-implementation-plan.md)
and the user's requirement that every provider resource use Platform's HTTP
abstraction. Milestones 10–17 remain **In Progress**.

## Source and ownership

- Provider baseline: merged PR 9, `739619d085185f7e3bc72270d650cf3263f0db34`.
- HTTP architecture plan: `b6a964a`.
- Provider implementation and qualified artifact:
  `b3db62a1be3df432e7049ff01120f961d0008707`.
- Independent consumer pin and checksums: `bb21bb4`.
- Platform `origin/main`, fetched during implementation:
  `4013cfd21d145dd5adf8f8e3382debdaeeefd93e` (PR 17). It still exposes the Transit-signed bucket metadata profile;
  no published public HTTP management abstraction for this provider is present.
- The current provider-owned schema remains `mock-v5`; no production routes or
  authentication scheme were invented or copied from the native wire protocol.

Platform owns service implementation, Transit authorization/protocols, internal
sequencing and durable backend recovery. This provider owns Terraform lifecycle,
its shared Go HTTP client, schema adaptation, tests and independent distribution.

## Delivered changes

- All 23 resources and 47 data sources use the shared Platform HTTP client. A
  protocol registry audit enumerates every surface and verifies real HTTPS
  requests, authentication, scope/version, routes and denial propagation; managed
  state is preserved. A production import guard prevents alternate service SDK,
  network, native, process or SQL paths from entering resource adapters.
- Removed the unregistered native package, binary vectors/capability fixtures and
  native packaging inputs. Historical source/results retain immutable links.
- Added an embedded HTTP contract identity shared by artifact metadata and release
  checks: version, transport, schema digest, provisional/published status and
  published Platform revision when available. Schema bytes are checked against it.
- Replaced native live configuration/report/workflow identities with HTTP API and
  schema identities, string scopes, Platform revision and exact artifact digest.
  Superseded native fields fail the configuration cutover. No credentials or
  service calls are used by the blocked prerequisite check.
- Publication requires a published source contract and exact matching artifact/
  report plus all sixteen advertised-surface cases. Synthetic test reports prove
  missing/duplicate/skipped/partial/provisional and mismatched evidence is refused.
- Regenerated reference text to describe Platform revisions and the common HTTP
  abstraction. The mock wire contract and resource schema stay at v5/version 3;
  HTTP routing and successful lifecycle behavior have not been renamed as a
  production API.

## Validation

Executed tuple: Go 1.26.7, Terraform 1.15.3, protocol 6, Linux amd64, mock-v5.
The installed suite, extracted archive and independent consumer all select the
exact committed artifact below. Source checks began on the implementation tree
before its commit; subsequent consumer and evidence edits do not alter the
provider executable sources.

| Check | Result and boundary |
| --- | --- |
| HTTP boundary audit | Passed: all 23 resources and 47 data sources reach the raw HTTPS peer; authentication, scope/version, route, denial and managed-state preservation verified. Production import boundary checks passed. |
| HTTP identity/live-report tests | Passed: schema identity, configuration cutover, explicit blocked preflight and rejection of incomplete/mismatched reports. |
| Python release tests | All six passed, including provisional-artifact refusal and complete published-contract case validation using synthetic test identities. |
| `just fmt`, `just build` | Passed. |
| `just check` | Passed formatting/workflow checks, six Python release tests, Nix install check, vet, race unit tests and all 37 top-level mock acceptance tests (1273.623 seconds for acceptance). |
| `nix flake check --all-systems --no-build --no-update-lock-file` | Passed evaluation on four systems; no non-Linux execution claim. |
| `just test-acc installed` | Passed all 37 top-level mock acceptance tests in 1245.004 seconds using the exact Nix mirror; separate-process cases have no compiler fallback. |
| Installed crash matrix | Both source-built and exact Nix artifacts passed 108 cases: 18 service kinds × create/update/delete × after acceptance/after completion, with killed provider processes and reloaded fixture ledgers. |
| Independent copied consumer | Passed outside the checkout: readonly checksums, ordinary Random provider, CRUD/import/empty plans and cleanup, with no compiler on PATH. |
| Extracted release ZIP | Exact-byte verification and installed graph/checksum/version/platform cases passed in 15.039 seconds. |
| Explicit HTTP preflight | Expected nonzero result; all 16 cases blocked without reading credentials or sending service requests. |
| Publication gate | Expected refusal for the actual provisional artifact; no release was signed or published. |
| Documentation | Generated references refreshed; relative links/fragments, roadmap numbering/phase membership/statuses and `git diff --check` passed. |

The [saved preflight report](platform-http-live-blocked.json) binds the selected
artifact, HTTP schema and reviewed Platform revision. Its blocked cases establish
prerequisite refusal only; they provide no real-service qualification.

## Exact candidate artifact

- Committed source: `b3db62a1be3df432e7049ff01120f961d0008707`.
- Version: `0.1.0-dev.gb3db62a1be3df432e7049ff01120f961d0008707`.
- Binary SHA-256: `64a00f60deb2c4a51c8b3535147c176b3f99c3a26ebcd4a9ff8b170ada580a94`.
- ZIP SHA-256: `4e10379324498fd41e35614209afd96d42f8a8548e780d6a6b1ce279adfd9171`.
- HTTP schema SHA-256: `a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b`.
- Contract identity: `mock-v5`, `https-json`, `provisional`; no published Platform
  contract revision is claimed by the executable.

The unsigned candidate ZIP uses the exact Nix mirror executable bytes. The copied
consumer pins that committed source in its flake and both locks. Documentation and
consumer-pin commits do not substitute a different artifact for this evidence.

## Corrections during validation

The first import-boundary run rejected the provider's pure `net/netip` parsing
dependency. The guard now permits that package and `net/url` while keeping network
I/O in the shared client; the corrected audit passed. Formatting was rerun after
staging native file deletions so Git's file inventory matched the working tree.
Updating the consumer intentionally required regenerating its Spoke checksum
entry from the trusted mirror; the Random lock entry was preserved and normal
consumer verification remained readonly.

## Remaining prerequisites

Platform must publish and implement its HTTP service contract and real backends.
The provider then reconciles the provisional schema/client and executes installed
metadata and full-service live cases, including actual data planes and restart
recovery. Current preflight correctly reports these gaps instead of native SDK
blockers. Publication remains unavailable; no milestone is completed by this work.
