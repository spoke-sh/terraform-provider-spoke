# Spoke Terraform Provider

[![CI](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/spoke-sh/terraform-provider-spoke/actions/workflows/ci.yml)

The Spoke Terraform provider lets you define and manage Spoke Platform resources
as code. Configure projects, access permissions, storage, workloads and networking
in Terraform, review changes with `terraform plan`, and apply them through
Platform's HTTP management API. Resources manage service configuration and
lifecycle, while data sources let you discover existing resources and inspect
their status.

**Status: experimental.** The provider is currently tested against a local test
service using the provisional mock-v5 API. Live Platform services are not yet
qualified, and no Terraform Registry release is available. Development versions
may introduce breaking configuration and state changes without migrations.

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
      version = "= 0.1.0-dev.gb3db62a1be3df432e7049ff01120f961d0008707"
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
| `just test-acc [mock/installed/live]` | Run mock acceptance, the installed Nix artifact, or the explicit live qualification gate. |

Mock tests use isolated local TLS services and need no Platform or cloud
credentials. The live gate requires explicit configuration and currently reports
missing Platform HTTP contract and real-service qualification prerequisites.
See the [HTTP boundary guide](docs/platform-http.md) and
[qualification record](qualification/platform-http.md) for details.

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
