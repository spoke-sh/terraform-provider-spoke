# Captured mock-v2 migration baseline

`mock-v2-state.json` and `mock-v2.ledger.json` were captured from the installed
provider built at `a97dfca` using Go 1.26.7 / Terraform 1.15.3 on Linux amd64.
The graph contains all five existing resource kinds, generated parent references,
an explicit grant and a bucket through a separate provider alias. Values and
credentials are synthetic. No live service was used.

The capture used the repository's installed-consumer harness and
`scopeGraphConfig`, with a bucket in the fixture's preexisting environment.
`TestAccMockV2StateMigration` consumes the checked-in state/ledger with the new
binary, using both derived and literal configured grant targets. It requires
empty plans and zero new fixture operations. The fixture migration separately
rejects pending old operations and preserves historical receipt fingerprints.

These files are historical evidence. Do not regenerate them with the new provider
to hide a migration regression.
