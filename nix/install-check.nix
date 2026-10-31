{
  runCommand,
  terraform,
  provider,
  mirror,
}:
runCommand "spoke-provider-install-check" { nativeBuildInputs = [ terraform ]; } ''
  export HOME="$TMPDIR/home" CHECKPOINT_DISABLE=1 TF_IN_AUTOMATION=1
  mkdir -p "$HOME"
  export TF_CLI_CONFIG_FILE="$TMPDIR/terraform.rc"
  cat > "$TF_CLI_CONFIG_FILE" <<EOF
  provider_installation {
    filesystem_mirror {
      path = "${mirror}"
      include = ["registry.terraform.io/spoke-sh/spoke"]
    }
    direct { exclude = ["registry.terraform.io/spoke-sh/spoke"] }
  }
  EOF
  cat > main.tf <<EOF
  terraform {
    required_version = "= ${terraform.version}"
    required_providers {
      spoke = { source = "spoke-sh/spoke", version = "= ${provider.version}" }
    }
  }
  EOF
  terraform providers lock -fs-mirror=${mirror} -platform=linux_amd64
  terraform init -backend=false -input=false -lockfile=readonly
  terraform providers schema -json > schema.json
  terraform validate
  mkdir -p "$out"
  cp .terraform.lock.hcl schema.json "$out/"
  cp ${provider}/share/spoke/identity.json "$out/"
''
