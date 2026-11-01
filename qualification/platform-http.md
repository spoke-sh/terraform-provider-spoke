# Platform HTTP boundary implementation

Implementation follows the revised [milestones 10–17 plan](../docs/roadmap/milestones-10-17-implementation-plan.md)
and the user's requirement that every provider resource use Platform's HTTP
abstraction. Milestones 10–17 remain **In Progress**.

## Source and ownership

- Provider baseline: merged PR 9, `739619d085185f7e3bc72270d650cf3263f0db34`.
- HTTP architecture plan: `b6a964a`.
- Platform `origin/main`, fetched during implementation:
  `4013cfd` (PR 17). It still exposes the Transit-signed bucket metadata profile;
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

Validation is in progress. This section will record executed commands and results
before handoff; no pending check is qualification evidence.

## Remaining prerequisites

Platform must publish and implement its HTTP service contract and real backends.
The provider then reconciles the provisional schema/client and executes installed
metadata and full-service live cases, including actual data planes and restart
recovery. Current preflight correctly reports these gaps instead of native SDK
blockers. Publication remains unavailable; no milestone is completed by this work.
