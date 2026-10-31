# Working on documentation

Follow [the repository instructions](../AGENTS.md).

`index.md`, `resources/` and `data-sources/` are generated from provider schemas
and examples. Change those sources and run `./scripts/generate-docs.sh` from the
repository root inside the Nix shell. Do not edit generated reference files by
hand or run tfplugindocs directly against `docs/`.

`roadmap/` and this file are hand-maintained. The generation script uses a
temporary output directory and replaces only the three generated paths above.
Extend that explicit list when adding a new generated reference category;
preserve other documentation. Roadmap scope and status follow its own
[instructions](roadmap/AGENTS.md).
