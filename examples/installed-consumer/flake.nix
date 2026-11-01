{
  description = "Independent checksum-locked Spoke provider consumer";
  inputs.provider.url = "github:spoke-sh/terraform-provider-spoke/b3db62a1be3df432e7049ff01120f961d0008707";
  inputs.nixpkgs.follows = "provider/nixpkgs";
  outputs =
    { provider, nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs {
        inherit system;
        config.allowUnfreePredicate = p: pkgs.lib.getName p == "terraform";
      };
      mirror = provider.packages.${system}.terraform-provider-spoke-mirror;
      runtime = [
        pkgs.terraform
        pkgs.bash
        pkgs.coreutils
        pkgs.jq
        provider.packages.${system}.mock-platform
      ];
      cli = pkgs.writeText "spoke-consumer.terraformrc" ''
        provider_installation {
          filesystem_mirror {
            path = "${mirror}"
            include = ["registry.terraform.io/spoke-sh/spoke"]
          }
          direct { exclude = ["registry.terraform.io/spoke-sh/spoke"] }
        }
      '';
    in
    {
      devShells.${system}.default = pkgs.mkShellNoCC {
        packages = runtime;
        TF_CLI_CONFIG_FILE = cli;
        CHECKPOINT_DISABLE = "1";
        TF_IN_AUTOMATION = "1";
        shellHook = ''export PATH="${pkgs.lib.makeBinPath runtime}"'';
      };
      packages.${system}.mirror = mirror;
    };
}
