{ runCommand, provider }:
runCommand "terraform-provider-spoke-mirror-${provider.version}"
  {
    inherit (provider) version;
  }
  ''
    destination="$out/registry.terraform.io/spoke-sh/spoke/${provider.version}/linux_amd64"
    mkdir -p "$destination"
    cp ${provider}/bin/terraform-provider-spoke_v${provider.version} "$destination/"
  ''
