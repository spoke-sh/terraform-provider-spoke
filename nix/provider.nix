{
  lib,
  buildGoModule,
  source,
  version,
  revision,
  nativeCaller,
  nativeRevision,
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
        (
          lib.hasPrefix "internal/" relative
          || lib.hasPrefix "cmd/mock-platform/" relative
          || lib.hasPrefix "cmd/native-journal/" relative
        )
        && lib.hasSuffix ".go" relative
      )
      || builtins.elem relative [
        "testdata/contracts/mock-v5.schema.json"
        "testdata/contracts/native-v2-vectors.json"
        "internal/nativecontract/schema.json"
        "internal/testutil/liveplatform/cases.json"
        "testdata/native-v2/vectors.json"
      ];
  };
  vendorHash = "sha256-ZIGpLI+GU7czKoGrJMFUmKwnbsi/u6V/aafTxt57lV0=";
  subPackages = [
    "."
    "cmd/native-journal"
  ];
  env.CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.revision=${revision}"
    "-X main.nativeRevision=${nativeRevision}"
    "-X github.com/spoke-sh/terraform-provider-spoke/internal/platformclient.ServingExecutable=${nativeCaller}/bin/transit-serving-client"
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
    description = "Spoke Terraform provider with explicit mock and native metadata profiles";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
  };
}
