# Publication declaration HTTP client qualification

This record qualifies the provider's Go HTTP client against Platform's installed,
retained declaration API. It does not qualify Terraform static resources, object
bytes, hosted delivery or a Registry release. See the
[HTTP guide](../docs/platform-http.md#publication-declaration-client) and
[milestone 18](../docs/roadmap/18-static-publication.md).

## Exact source and executable evidence

The [saved report](publication-client.json) was produced at
2026-11-02 23:30 UTC from clean source checkouts. It records six passing scenario
groups in 9.143 seconds, including actual caller SIGKILL. Build/setup time is
outside that duration. Source identities and binary SHA-256 values are retained
in the report; documentation commits do not relabel a newer executable as tested.

| Input | Identity |
| --- | --- |
| Provider client/test source | `67c9a7c52338ed5ed41c90d197b1f9eedae66b67` |
| Go race test executable SHA-256 | `7f9abcf6d7fc2b846acfa52609a39151a5a7b0260398611e60f026f24d2aa4b8` |
| Platform installed fixture source | `c56f82d41ec9b4fc3617f732e2c900773cc00080` |
| Contract source | `6627bed1179b7cb95f51831d26d35e207c65481f` |
| Schema SHA-256 | `db04fc4ce0e14534c2e44cef932b197e6d9aeb48006f21f7a1523c21def0511f` |
| Wire/profile | `publication-v1` / `spoke.platform.http.publication.v1` |
| System/toolchain | Linux amd64, Go 1.26.7, locked provider and Platform Nix shells |

The contract schema and copied profile are byte-identical between their pinned
source and the fixture revision. [Platform PR #40](https://github.com/spoke-sh/platform/pull/40)
was open during execution; main `71d43c1` did not yet contain this API. Its separate
native/HTTP report and CI results are upstream evidence, not this client's results.

## Executed behavior

- Application policy create/read/update/delete and exact original-key replay;
  conflicting reuse is rejected. Independent client drift is visible and stale
  revision fences fail before an explicitly new intent converges the declaration.
- Project binding reservation, environment target creation, target policy changes
  and target retirement with an observed project hold release. Policy changes
  preserve runtime selection; readiness remains pending and endpoints remain null.
- Reader, foreign-principal and wrong-environment denial; protected application
  deletion and deletion while an environment binding holds the parent are refused.
- Complete application, binding and target discovery plus receipt history spanning
  more than one real page. Target absence after deletion is an authoritative 404.
- An independent Go caller settles a create, reaches a test barrier before saving
  caller state, and is actually killed. A new process uses the same key and HTTP
  intent; the fixture independently verifies that the retained operation ID is
  unchanged. This is caller recovery, not a Terraform process or native-node crash.
- The client-only HTTPS conformance suite separately tests uncertain acceptance,
  malformed/ambiguous JSON, required/null fields, counter bounds, profile/principal
  changes, incomplete/duplicate/changing collections, history filters, tombstones,
  ownership/receipt mismatches, denial, safe diagnostics and all eight mutation
  routes. These synthetic peer results are distinct from the native run above.

Native setup, scope/role installation, signing and activation belong to Platform's
fixture. The provider supplies only its normal HTTP client. Loopback TLS and fresh
synthetic token files are private and outside Nix; no ambient service credentials
or cloud infrastructure are used. Reports contain no credential values.

## Validation

| Check | Result / scope |
| --- | --- |
| Focused publication tests with race detector | Passed, including state-filter and expiry cases |
| Six release gate tests | Passed; client/metadata evidence does not enable publication |
| Explicit installed Platform client fixture | Passed; six groups, exact identities above |
| `just fmt`, `just build` | Passed |
| All-system Nix evaluation without builds or lock updates | Passed; not non-Linux runtime qualification |
| Documentation generation and local links/status/badge checks | Passed; 10 Markdown files, 101 local links, 19 milestone statuses and three upstream badges; generated reference is unchanged |
| Candidate ZIP construction and verification | Passed; exact mirror bytes, including publication profile/schema provenance; no publication |
| `just check` | Passed; formatting, workflows, flake, vet, race unit tests and 37 top-level mock acceptance tests (acceptance: 1,057.465 seconds) |
| `just test-acc installed` | Passed; 37 top-level acceptance tests against the exact Nix artifact below (1,038.640 seconds), including separate-process recovery |

The installed regression candidate is
`0.1.0-dev.g67c9a7c52338ed5ed41c90d197b1f9eedae66b67`, Linux amd64, binary
SHA-256 `6e7003f3c429a662fc1e0aca02360ac43a21ff5a44149d3f5ccf34d52bd63048`.
This is a separate executable from the Go client test above. Its regression
fixture exercises the existing Terraform surface; it cannot qualify unregistered
static Terraform resources.

## Remaining gates

No static Terraform resource/data source is registered. Caller-supplied operation
keys in this client do not alone solve Terraform's persisted intent/ownership
model. The lifecycle adapter must establish recoverable keys before dispatch,
including update/delete, import, replacement and A → B → A, without transferring
ownership or restoring a runtime-selected head on refresh.

Installed Terraform static lifecycle, immutable object upload/sealing, preview
and stable-domain delivery, CDN/TLS behavior and the full service/release gate
remain unexecuted. The existing independent bucket consumer pin is unchanged.
Neither mock regression results nor this Go client fixture satisfy those gates.
