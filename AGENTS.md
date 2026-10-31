# Developing the Spoke Terraform provider

## Scope and source of truth

- Read `README.md` and `HANDOFF.md` before implementation. Follow the
  [provider implementation plan](https://github.com/spoke-sh/terraform-provider-spoke/issues/1),
  which carries forward Transit issue #290 with corrected ownership and release
  versioning and adds development against mocked service endpoints.
- This repository owns the Go provider, platform API client integration, tests,
  examples, generated documentation, Nix packaging, and independent releases.
  Platform service APIs and provisioning belong in `spoke-sh/platform`; generic
  runtime primitives belong in `spoke-sh/transit`.
- Keep implementation planning in GitHub issues. Check the destination
  repository's instructions before making cross-repository changes.
- The current provider implements bucket lifecycle and scoped discovery against
  the provisional mock-v1 API. Do not report mock tests or skipped checks as
  qualified live services.

## Environment and everyday commands

Use the locked Nix shell: `nix develop` or `nix develop .#terraform`. For a single
command, use `nix develop --command just check`.

Keep the justfile at **no more than five public project tasks**, plus the private
default recipe that lists them. Extend the existing tasks rather than adding
another public command:

| Command | Use |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform, and Go files. |
| `just check` | Check formatting, workflows, and the flake; run Go vet, unit tests, and mock Terraform acceptance. |
| `just build` | Build the root provider executable into `bin/`. |
| `just test` | Run unit tests with the race detector and `TF_ACC` unset. |
| `just test-acc [mock]` | Run real Terraform with local TLS mock endpoints and synthetic credentials. Live mode currently fails explicitly. |

Bare `just` lists the tasks. Missing source, Terraform, or mock acceptance cases
must fail checks. Never select live tests from ambient credentials or silently
fall back from live tests to mocks.

Keep the nixpkgs revision and `flake.lock` pinned. Update them deliberately and
verify shell entry after toolchain changes. Go libraries belong in `go.mod` and
`go.sum`; keep `GOTOOLCHAIN=local` so builds use the Nix toolchain. Nix evaluation,
builds, and shell entry must have no deployment effects or embedded credentials.

## Provider implementation

- Use Go Terraform Plugin Framework and protocol 6. Root `main.go` serves the
  provider; `internal/provider/` owns Terraform schemas/lifecycle and
  `internal/platformclient/` owns the API boundary. Keep test-only service
  behavior in `internal/testutil/mockplatform/`, contract fixtures in
  `testdata/contracts/`, examples in `examples/`, and generated docs in `docs/`.
- Use the published platform management contract where available. Until it
  exists, develop against the plan's versioned, explicitly provisional test
  contract and stateful HTTP mock. Exercise the actual provider/client code;
  keep mock-only resources experimental until qualified against the real API.
  Prefer a thin Go HTTP client and schema-derived boundary types. Add a shared
  native client only if the selected protocol requires it.
- Keep plan, refresh, and data-source reads observational. Do not provision
  services by writing private platform tables or invoking cloud CLIs, SQL,
  migrations, or Nix from the provider.
- Use stable scoped IDs, explicit target selection, verified TLS, bounded
  timeouts, and revision preconditions. Distinguish accepted operations from
  completed work. Establish recoverable request identity before mutation;
  uncertain retries must inspect or resume the original operation.
- Remove state only on authoritative absence. Denied reads, outages, and
  incomplete listings are errors. Use dependency-restricted deletion and
  explicit data protection; preserve separately owned data and artifacts.
- Keep credentials and secret values outside Terraform state, plans,
  diagnostics, logs, and the Nix store. A sensitive schema flag alone does not
  prevent values from being stored.
- Preserve the MIT license. Keep examples and generated resource documentation
  aligned with implemented behavior. After schema or example changes, run
  `tfplugindocs generate --provider-name spoke` from the Nix shell.

## Validation and CI

Run `just fmt`, `just check`, and `just build` before handing off code changes.
For Nix changes, also run
`nix flake check --all-systems --no-build --no-update-lock-file`; evaluation on
other systems does not establish that their binaries build or run.

Add focused tests for changed provider behavior, including null/unknown values,
import, drift, authorization failures, revision conflicts, protected deletion,
and interrupted operations where relevant. Real lifecycle acceptance requires
an explicitly configured isolated backend; ordinary checks must run without
service credentials, production state initialization, or provisioning effects.

`.github/workflows/ci.yml` runs the check and build tasks on Linux, including
mock acceptance and Linux process-crash recovery. Keep GitHub Actions pinned to
verified commit SHAs with
version comments, read-only repository permissions, and no persisted checkout
credentials. Keep real acceptance tests outside this ordinary CI workflow.

Summarize what changed, which checks ran, and any skipped or unqualified behavior.
Do not claim cross-platform or real-service support from local unit tests alone.
