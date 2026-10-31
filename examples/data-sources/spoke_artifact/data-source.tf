# Build from pinned source/Nix inputs and upload/promote separately first.
# A local Nix store path is not an image digest or an upload operation.
data "spoke_artifact" "release" {
  registry_id = "registry-example"
  repository  = "application/web"
  digest      = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
# Pass data.spoke_artifact.release.resolved_digest to the container declaration.
# Tag selection is also available, but its result may change on a later read.
