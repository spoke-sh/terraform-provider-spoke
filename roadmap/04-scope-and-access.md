# 4. Scope and access

[Roadmap](README.md) · **Status:** Completed

Let a customer declare project and environment ownership, machine identities and
explicit grants before creating dependent services.

## Prerequisites and scope

Use [management contracts](02-management-contracts-and-discovery.md) and the
[recovery gates](03-recovery-isolation-and-protection.md). Project/environment
data sources remain observational; separate resources now manage those scopes.
Platform owns tenancy and policy enforcement. Credential issuance and rotation
remain external to Terraform state.

## Delivered implementation and evidence

`spoke_project`, `spoke_environment`, `spoke_service_account` and `spoke_grant`
implement lifecycle/import with explicit scopes, principal ownership and revision
fences. Organization-only provider configuration creates projects and dependent
environments without unknown provider targets. All five managed collections have
bounded read-only data sources. Generated reference and bootstrap/service examples
are linked from the [current usage guide](../README.md).

[Executed qualification](../qualification/milestones-2-4.md) at `32d6894` covers
the combined Terraform graph, each resource's lifecycle, import, empty plans,
rename/drift, protection, lost replies and authoritative absence; client tests
cover interrupted CRUD and fixture reload for every new kind. Installed consumers
prove parent-safe bootstrap, saved-plan revision conflicts for all four kinds,
self-revocation state retention and provider aliases. Independent grant revocation
is verified through the actual HTTP client. These are mock-v2 guarantees; real
platform tenancy and policy remain milestone 10.

## Delivery sequence

1. Specify organization-, project- and environment-level API scopes, canonical
   import IDs, parent relationships, immutable fields and deletion restrictions.
   Extend the fixture with independent tenants and authorization checks.
2. Resolve bootstrap configuration before adding resources. The initial provider
   required an existing project and environment. Use organization-only bootstrap
   and explicit project/environment assertions or provider aliases so managing a
   parent never requires configuring the provider with that parent's unknown ID.
   Do not bootstrap the platform management service with this provider.
3. Implement `spoke_project` and `spoke_environment` lifecycle and exact-ID reads,
   preserving parent identity on rename and rejecting mismatched target scope.
4. Implement `spoke_service_account` and `spoke_grant` with attributable subjects,
   exact targets and explicit permissions. A binding or successful login does
   not create a grant. Delete only the grant this resource owns; independent
   grants and owners must survive.
5. Define self-revocation and lost inspection rights. An apply that loses access
   retains uncertainty and recovery context; it must not erase or recreate
   resources it can no longer inspect. Document external credential management.

## Acceptance criteria

- Real Terraform can create a project and environment from an already available
  organization-scoped endpoint without a configuration dependency cycle; a
  subsequent service provider configuration selects the created scope explicitly.
- Every resource passes lifecycle, import, empty-plan, rename/drift, stale
  revision, interrupted operation and authoritative-absence tests.
- Cross-tenant reads/mutations fail; revocation takes effect through the same
  policy owner and denied refresh preserves state. Independent grants survive
  removal of this Terraform-managed grant.
- Parent deletion with children is refused without cascade. A controlled teardown
  removes only owned children and preserves separately managed resources.
- Schema, state, plans and failure diagnostics contain no generated credentials.
  Documentation identifies the draft API and exact tested permission behavior.
