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
        "testdata/contracts/mock-v4.schema.json"
        "testdata/contracts/native-v1-vectors.json"
      ];
  };
  vendorHash = "sha256-9a4TDOtr/IVrV/zHl1eQLGrEomh/9Vl4lBJ1iC2Ha8c=";
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
    description = "Experimental Spoke Terraform provider (mock contract only)";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
  };
}
