{
  lib,
  buildGoModule,
  source,
  version,
  revision,
}:
buildGoModule {
  pname = "terraform-provider-spoke";
  inherit version;
  src = lib.cleanSourceWith {
    src = source;
    filter =
      path: type:
      let
        relative = lib.removePrefix (toString source + "/") path;
      in
      type == "directory"
      || builtins.elem relative [
        "go.mod"
        "go.sum"
        "main.go"
        "LICENSE"
        "VERSION"
      ]
      || (
        (lib.hasPrefix "internal/" relative || lib.hasPrefix "cmd/mock-platform/" relative)
        && lib.hasSuffix ".go" relative
      )
      || builtins.elem relative [
        "testdata/contracts/mock-v5.schema.json"
        "internal/platformclient/contract.json"
        "internal/platformclient/profile.json"
        "internal/platformclient/publication-contract/profile.json"
        "internal/platformclient/publication-contract/schema.json"
        "internal/platformclient/governor-contract/profile.json"
        "internal/platformclient/governor-contract/schema.json"
        "internal/platformclient/governor-contract/vectors.json"
        "internal/platformclient/governor-contract/upstream-profile.md"
        "internal/testutil/liveplatform/cases.json"
      ];
  };
  # Governor list validation adds packages from the already locked validator module.
  vendorHash = "sha256-t51f2VqSqD6kFJbBUeQB0HweS4DwlVc9SQxzHWeGT/0=";
  subPackages = [ "." ];
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.revision=${revision}"
  ];
  # The separate credential-free CI lane runs race and Terraform acceptance.
  # Keep the package sandbox bounded and free of deployment effects.
  doCheck = false;
  postInstall = ''
    mv "$out/bin/terraform-provider-spoke" "$out/bin/terraform-provider-spoke_v${version}"
    mkdir -p "$out/share/spoke"
    "$out/bin/terraform-provider-spoke_v${version}" -version > "$out/share/spoke/identity.json"
    cp LICENSE "$out/share/spoke/LICENSE"
  '';
  meta = {
    description = "Experimental Spoke Terraform provider with Platform HTTP metadata integration";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
  };
}
