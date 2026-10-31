# Developing the Spoke Terraform provider

## Scope and source of truth

- Read `README.md` and `HANDOFF.md` before implementation. The detailed plan is
  [Transit issue #290](https://github.com/spoke-sh/transit/issues/290); apply the
  handoff's corrections to repository ownership and release versioning.
- This repository owns the Go provider, platform API client integration, tests,
  examples, generated documentation, Nix packaging, and independent releases.
  Platform service APIs and provisioning belong in `spoke-sh/platform`; generic
  runtime primitives belong in `spoke-sh/transit`.
- Keep implementation planning in GitHub issues. Check the destination
  repository's instructions before making cross-repository changes.
- The current checkout contains development tooling, not an implemented
  provider. Do not report mocked resources or skipped checks as working services.

## Environment and everyday commands

Use the locked Nix shell: `nix develop` or `nix develop .#terraform`. For a single
command, use `nix develop --command just check`.

Keep the justfile at **no more than five public project tasks**, plus the private
default recipe that lists them. Extend the existing tasks rather than adding
another public command:

| Command | Use |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform, and Go files. |
| `just check` | Check formatting, workflows, and the flake; run Go vet and unit tests when the module exists. |
| `just build` | Build the root provider executable into `bin/`. |
| `just test` | Run unit tests with the race detector and `TF_ACC` unset. |
| `just test-acc` | Explicitly run real provider acceptance tests using an isolated fixture and external credentials. |

Bare `just` lists the tasks. Until `go.mod` exists, `check` explicitly skips the
provider checks, while build and test tasks fail with a missing-scaffold message.
Remove this transitional behavior when the scaffold is established.

Keep the nixpkgs revision and `flake.lock` pinned. Update them deliberately and
verify shell entry after toolchain changes. Go libraries belong in `go.mod` and
`go.sum`; keep `GOTOOLCHAIN=local` so builds use the Nix toolchain. Nix evaluation,
builds, and shell entry must have no deployment effects or embedded credentials.

## Provider implementation

- Use the Go Terraform Plugin Framework and protocol 6. Start with a root
  `go.mod`/`go.sum` and `main.go`, then `internal/provider/` for Terraform schemas
  and lifecycle handling and `internal/platformclient/` for the API boundary.
  Add `examples/` and generated `docs/` as resources become available.
- Verify the supported platform management contract before implementing a
  resource. Prefer a thin Go HTTP client and schema-derived boundary types.
  Add a shared native client only if the selected protocol requires it.
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
  aligned with implemented behavior; use `tfplugindocs` from the Nix shell.

## Validation and CI

Run `just fmt` and `just check` before handing off changes. Run `just build` when
provider code exists. For Nix changes, also run
`nix flake check --all-systems --no-build --no-update-lock-file`; evaluation on
other systems does not establish that their binaries build or run.

Add focused tests for changed provider behavior, including null/unknown values,
import, drift, authorization failures, revision conflicts, protected deletion,
and interrupted operations where relevant. Real lifecycle acceptance requires
an explicitly configured isolated backend; ordinary checks must run without
service credentials, production state initialization, or provisioning effects.

`.github/workflows/ci.yml` runs the local check task and, once scaffolded, the
build task on Linux. Keep GitHub Actions pinned to verified commit SHAs with
version comments, read-only repository permissions, and no persisted checkout
credentials. Keep real acceptance tests outside this ordinary CI workflow.

Summarize what changed, which checks ran, and any skipped or unqualified behavior.
Do not claim cross-platform or real-service support from local unit tests alone.
