# Spoke Terraform Provider

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)
[![Platform UX review](https://github.com/spoke-sh/platform/actions/workflows/ux-review.yml/badge.svg?branch=main&event=push)](https://github.com/spoke-sh/platform/actions/workflows/ux-review.yml?query=branch%3Amain+event%3Apush)
[![Transit transaction recovery (source)](https://github.com/spoke-sh/platform/actions/workflows/native-transit-adoption.yml/badge.svg?branch=main&event=push)](https://github.com/spoke-sh/platform/actions/workflows/native-transit-adoption.yml?query=branch%3Amain+event%3Apush)
[![Platform build and integration](https://github.com/spoke-sh/platform/actions/workflows/check.yml/badge.svg?branch=main&event=push)](https://github.com/spoke-sh/platform/actions/workflows/check.yml?query=branch%3Amain+event%3Apush)

The Spoke Terraform provider lets you define and manage Spoke Platform resources
as code. Configure projects, access permissions, storage, workloads and networking
in Terraform, review changes with `terraform plan`, and apply them through
Platform's HTTP management API. Resources manage service configuration and
lifecycle, while data sources let you discover existing resources and inspect
their status.

**Status: experimental.** Provider CI uses the provisional mock-v5 test service.
Platform also records local installed-provider tests against its Transit-backed
HTTP bucket-declaration API; this manages metadata, not usable object storage.
Live services are not yet qualified, and no Terraform Registry release is
available. Development versions may introduce breaking configuration and state
changes without migrations.

The Platform badges follow upstream `main` pushes: automated UX checks,
transaction recovery against a source-built Transit candidate, and build/integration
checks (including the installed bucket provider, publication API and object feasibility).
A passing feasibility check can still report blocked adoption; these badges do
not establish live-service support. The [Platform adoption plan](docs/roadmap/platform-adoption-implementation-plan.md)
records reviewed revisions, failures, remaining gates and the proposed next work.

The separate [publication declaration HTTP client](docs/platform-http.md#publication-declaration-client)
is locally qualified against Platform's API. The experimental
[application](docs/resources/static_application.md),
[reservation](docs/resources/static_target_binding.md) and
[target](docs/resources/static_target.md) resources manage declaration policy with
[retained operation recovery](docs/publication-identity.md). Project and environment
roots keep separate credentials and state; runtime heads remain observational.
The [independent consumer](examples/installed-consumer/README.md#independent-publication-declarations)
pins the qualified artifact and readonly checksums; the explicit
[declaration gate](docs/platform-http.md#publication-declaration-gate) checks the
normal API separately from hosted services. Uploads and website delivery remain
Platform prerequisites.

The experimental [CI governor](docs/resources/ci_governor.md) manages operator
policy through the separately pinned provisional `ci-governor-v1` HTTP profile,
with [exact](docs/data-sources/ci_governor.md) and
[collection](docs/data-sources/ci_governors.md) discovery. Its
[identity protocol](docs/governor-identity.md) preserves original requests across
Terraform process death. [Qualification](qualification/governor-terraform.md)
is limited to installed Terraform against synthetic HTTPS peers; native service
management, physical workers, warming, handoff and billing remain unqualified.
Customers use the workflow in
[`spoke-github-action`](https://github.com/spoke-sh/spoke-github-action) without
declaring capacity or caches. The [roadmap](docs/roadmap/ci-governor-implementation-plan.md)
retains the separate service and release gates.

## Usage

Install through the [Nix filesystem mirror](docs/distribution.md#nix-installation).
The [installed consumer example](examples/installed-consumer/README.md) provides
a runnable local walkthrough with a pinned provider, checksums and test service.
The tested combination is Terraform 1.15.3 on Linux amd64.

After configuring the mirror, a bucket configuration looks like this. The version
below matches the installed consumer; replace the endpoint, existing scope IDs
and credential path with those for your test environment.

```terraform
terraform {
  required_version = "= 1.15.3"
  required_providers {
    spoke = {
      source  = "spoke-sh/spoke"
      version = "= 0.1.0-dev.gbc1860930924ee5c0a88004273eb85bedf54cfe2"
    }
  }
}

provider "spoke" {
  endpoint        = "https://management.example.invalid"
  organization_id = "org-example"
  project_id      = "project-example"
  environment_id  = "env-example"
  token_file      = pathexpand("~/.config/spoke/token")
}

resource "spoke_bucket" "assets" {
  management_key      = "application-assets"
  name                = "assets"
  capacity_bytes      = 1073741824
  retention_days      = 30
  deletion_protection = true
}
```

Credentials come from an external `token_file`, or from `SPOKE_TOKEN` when that
attribute is omitted. Add `ca_file` when your endpoint uses a custom certificate
authority; TLS verification remains enabled. Credential values stay outside
Terraform configuration and state.

Keep each resource's `management_key` unique and stable so interrupted operations
can recover their original identity. Before destroying the example bucket, apply
`deletion_protection = false`; nonempty storage and dependent resources can still
prevent deletion. A successful apply records the service declaration; inspect
`readiness` and other status attributes for runtime availability.

## Documentation and examples

- [Provider configuration](docs/index.md), [resources](docs/resources/) and
  [data sources](docs/data-sources/) cover the available settings and import IDs.
- [Bootstrap projects and environments](examples/bootstrap/main.tf), then
  [manage services in an existing scope](examples/scoped-service/main.tf).
- [Deploy an application graph](examples/service-graph/README.md) with a registry,
  database, bucket, secret reference, runtime grants, container and endpoint.
  Image uploads and secret issuance are handled outside Terraform.
- [Installation and distribution](docs/distribution.md) explains mirrors,
  version pins and checksums.

## Development

Install Nix with flakes enabled, then enter the pinned development environment:

```sh
nix develop
just
```

`nix develop .#terraform` selects the same shell. It includes Go, Terraform and
the repository's formatting, documentation and test tools. Bare `just` lists the
five project tasks:

| Task | Purpose |
| --- | --- |
| `just fmt` | Format Nix, Just, Terraform and Go source. |
| `just check` | Check formatting, workflows and the flake; run Go vet, race unit tests and mock Terraform acceptance. |
| `just build` | Build `bin/terraform-provider-spoke`. |
| `just test` | Run unit tests with the race detector. |
| `just test-acc [mock/installed/live]` | Run mock acceptance, the installed Nix artifact, or explicitly configured Platform metadata/declaration/service qualification. |

Mock tests use isolated local TLS services and need no Platform or cloud
credentials. The live lane can execute the pinned local Platform bucket-metadata
or publication declaration fixture using an exact installed provider. Hosted
service selections still report unmet prerequisites. Platform's narrower development bucket profile does
not satisfy the provider's full live/release gate.
See the [HTTP boundary guide](docs/platform-http.md) and
[qualification record](qualification/platform-adoption.md) for details.

Read [AGENTS.md](AGENTS.md) for development conventions and required checks.
The [roadmap](docs/roadmap/README.md) tracks remaining work and implementation
plans. Keep dependency lockfiles committed and update their pins deliberately.

After changing schemas or examples, regenerate the reference documentation:

```sh
nix develop --command ./scripts/generate-docs.sh
```

The script preserves the hand-maintained roadmap.
[CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)
evaluates the declared Nix systems and runs checks, builds and mock acceptance
on Linux. Live qualification uses a separate workflow.
