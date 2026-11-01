# Working on the roadmap

Follow [the repository instructions](../../AGENTS.md). Read the [index](README.md)
and the affected milestone before changing scope or status.

Mirror Transit's planning structure: numbered milestone files, a phase/status
index, explicit prerequisites, delivery sequence and executable acceptance
criteria. Keep current behavior, delivered evidence and remaining work separate.
Do not copy Transit runtime requirements into this provider's ownership. The
provider is a Go HTTP client of Platform's public management API; Platform owns
Transit integration and internal sequencing. Do not plan provider-side native
SDK/ABI dependencies, signers or signed-envelope journals. Preserve stable logical
request identity and server-backed operation recovery through HTTP, including
provider death before Terraform saves state.

Use `Planned`, `In Progress` or `Completed` in both each header and the index.
Keep numbering contiguous and phase membership exhaustive and unique. Update
links, dependency references and checkpoints together when moving work. Preserve
completed files and prior evidence; reopen unmet criteria rather than silently
reducing their scope.

Every resource milestone must specify actual Terraform caller behavior,
authorization, bounded work, uncertainty/recovery and deletion protection.
Separate mock evidence, installed-provider evidence and real service evidence.
An unimplemented or unqualified backend is an explicit prerequisite, not a reason
to invent a supported endpoint or mark a mock as production support.

Keep user instructions and generated schemas in their existing owners. This
directory is hand-maintained and must survive schema documentation generation;
use [`scripts/generate-docs.sh`](../../scripts/generate-docs.sh). Before handing
off documentation changes, check relative links and fragments, numbering,
dependency order, index/header statuses and `git diff --check`. Code tests are
needed only if executable behavior also changes.

Until all milestones 1–17 are complete, plan hard contract cutovers without
backwards compatibility. Do not add legacy mock/state migration gates. Preserve
historical completion evidence and current-contract recovery/data protection.
Milestone 11 qualifies current CLI/API/system combinations and immutable artifacts;
it does not introduce a historical compatibility promise before that gate.
