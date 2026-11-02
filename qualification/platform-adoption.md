# Platform bucket profile adoption

This is the completed bucket-profile record. Subsequent publication client work
has its own [execution record](publication-client.md) and advances milestone 18
to In Progress without changing the bucket artifact evidence below.

This record separates provider conformance, mock acceptance and real local
Platform metadata. It does not qualify object storage, hosted services or public
release. The [implementation plan](../docs/roadmap/platform-adoption-implementation-plan.md)
and [HTTP guide](../docs/platform-http.md) describe the selected boundaries.

## Implemented changes

- Independently pinned Platform `spoke.platform.http.bucket.v1` schema/vectors,
  explicit executable identity, profile binding and input/observation bounds.
  The complete mock-v5 inventory remains provisional.
- Typed complete owner-filtered pending discovery, with denial, unknown profile,
  malformed/partial response and bounded transient-read tests. Reads remain
  observational and uncertain writes keep their original management identity.
- An explicit `local-platform-metadata` selection in `just test-acc live` executes
  the Platform-owned retained HTTPS fixture and installed Terraform consumer.
  It validates clean committed source, exact binary and contract pins, fresh
  reports and every required schedule. Ambient credentials/overrides are excluded.
- Reports label local metadata and cannot satisfy the complete service publication
  gate. Hosted service selections still fail on their unmet prerequisites.
- Platform companion changes recover uncertain concurrency cleanup by inspecting
  the original receipt before any identical replay, test an uncertain DELETE
  after acceptance with no second write, and exercise installed external drift.
  Its runner can consume an exact installed binary and exports fresh evidence.

## Exact installed tuple

| Input | Identity |
| --- | --- |
| Provider code/artifact | `67682f0af9e14cdf9fbc4d1b76a59525b2a509ad` |
| Provider version | `0.1.0-dev.g67682f0af9e14cdf9fbc4d1b76a59525b2a509ad` |
| Binary SHA-256 | `bd29d12d0c5eacf60bef716c64b99dde985a246c717653dabae008745c5016e8` |
| Platform fixture | `e40a1f49940e122ccbd9dd23bb69afa1416670ee` |
| Profile source | `ebf97fe086a99ba91f6b669b9f922f3964eb5bfa` |
| Profile schema SHA-256 | `8568a6483a450b79464386d9cd6e5be5c7ad0c4163ceecd0ffe3a7682b4341bd` |
| Fingerprint vectors SHA-256 | `48367b59715053a70a40979ddb57f615dcd9e2d38b29923cc48f14d948a1fcc3` |
| Provider inventory SHA-256 | `a3100081694a125dee8779179d1a85fb3f5ff1cdd8af6f7a8dab46dd35333c0b` |
| CLI/toolchain/system | Terraform 1.15.3, Go 1.26.7, Linux amd64, protocol 6 |
| Platform runtime | Transit `94fa12ff6d8cd3e409ac6524f721a17d1a245abb`; owned and installed by Platform |

The provider runner's subsequent evidence-directory correction affects only the
qualification harness, not the selected provider artifact. It retains private logs
outside Nix's temporary shell directory. Documentation commits do not substitute
a newer binary for the exact artifact above. The independent consumer now pins
this same `67682f0` artifact in its flake, exact provider constraint and both locks;
its copied execution remains a synthetic service check, separate from Platform.

## Execution and validation

The [saved metadata report](platform-adoption-metadata.json) contains the exact
provider/Platform identities and five passing metadata case groups. The underlying
Platform run executes independent raw HTTPS plus installed CRUD/import/empty plans,
external drift, rename, protection, owner/revision denial, quota and signer races,
token rotation, retained restart, credential canaries and four process-death/expiry
schedules. Terraform runs from a checksum-locked filesystem mirror without a
compiler in PATH. No mock or in-process provider replaces the Platform service.

| Check | Result / scope |
| --- | --- |
| Independent profile vectors and focused Go client tests | Passed |
| Metadata configuration/report validation, detached-consumer deadline recovery and six release gate tests | Passed; development/partial/mismatched evidence cannot publish |
| `just fmt`, `just build` | Passed |
| `just check` | Passed; formatting, workflows, flake, vet, race unit tests and 37 top-level mock acceptance tests (acceptance: 1,018.911 seconds) |
| `just test-acc installed` | Passed; the same 37 top-level acceptance tests against the exact artifact above (975.905 seconds) |
| Explicit configured `just test-acc live` metadata | Passed twice; latest run 49.227 seconds, real local retained Transit-backed bucket declarations |
| Independent consumer copied outside the checkout | Passed against the pinned artifact with readonly checksums and no compiler in Terraform's PATH |
| Documentation generation and review | Passed; 13 Markdown files, 107 local links/anchors, 19 milestone statuses and phase membership, badge paths/filters and whitespace |
| All-system Nix evaluation without builds/lock updates | Passed; this is not non-Linux runtime qualification |
| Platform companion `just check` | Passed, including Terraform validation and 11 offline infrastructure cases |
| Hosted services, other data planes, Registry/release | Not executed; prerequisites remain unavailable |

The prior Platform main failure at
[run 41397981615](https://github.com/spoke-sh/platform/actions/runs/41397981615)
remains failed evidence. A fresh original-source run passed locally, so that CI
failure is intermittent. The failed assertion assumed cleanup DELETE must return
202 although the contract permits transient uncertainty. The companion harness
now bounds recovery by original key/generation; its deterministic post-acceptance
case proves the original DELETE is recovered without another write. This is not
proof that native capacity failures are eliminated or replicated gateways qualify.

## Remaining gates

Platform's usable object/database/OCI/workload and wider APIs are still required
for TF10–17 service qualification and release. Metadata declares intent with
pending readiness and no object endpoint. Native owner replacement and browser
command delegation remain separate Platform work. No live credentials or cloud
resources were used, no release was published, and no milestone is closed here.

Static publication and model resources now have separate Planned milestones
18–19. Their management schemas and actual service backends are prerequisites;
no invented Terraform resource is registered. The original 1–17 compatibility
policy and TF10 initial graph remain unchanged. No visible UI changed, so UX
capture is not applicable to either implementation.
