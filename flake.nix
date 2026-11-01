{
  description = "Development environment for the Spoke Terraform provider";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/1e8bc658fc985ef27ccd66d107d767b32bb7ef98";
  inputs.transit-sdk.url = "github:spoke-sh/transit/c2e50061fc8622300e3a517e6e3a82f5c6eb6db1";

  outputs =
    {
      self,
      nixpkgs,
      transit-sdk,
      ...
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      eachSystem = nixpkgs.lib.genAttrs systems;
      pkgsFor =
        system:
        import nixpkgs {
          inherit system;
          config.allowUnfreePredicate = pkg: nixpkgs.lib.getName pkg == "terraform";
        };
    in
    {
      packages.x86_64-linux =
        let
          pkgs = pkgsFor "x86_64-linux";
          revision =
            self.rev
              or "dirty-${builtins.substring 0 16 (builtins.hashString "sha256" (toString self.outPath))}";
          version = "${pkgs.lib.removeSuffix "\n" (builtins.readFile ./VERSION)}-dev.g${revision}";
          provider = pkgs.callPackage ./nix/provider.nix {
            source = self;
            nativeCaller = transit-sdk.packages.x86_64-linux.transit-serving-client;
            nativeRevision = transit-sdk.rev;
            inherit version revision;
          };
          mirror = pkgs.callPackage ./nix/mirror.nix { inherit provider; };
        in
        {
          default = provider;
          terraform-provider-spoke = provider;
          terraform-provider-spoke-mirror = mirror;
          mock-platform = provider.overrideAttrs (_: {
            pname = "spoke-mock-platform";
            subPackages = [ "cmd/mock-platform" ];
            postInstall = "";
          });
        };

      checks.x86_64-linux.provider-install =
        (pkgsFor "x86_64-linux").callPackage ./nix/install-check.nix
          {
            provider = self.packages.x86_64-linux.terraform-provider-spoke;
            mirror = self.packages.x86_64-linux.terraform-provider-spoke-mirror;
          };

      devShells = eachSystem (
        system:
        let
          pkgs = pkgsFor system;
          # Include a C compiler for Go's race detector.
          shell = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              gotools
              terraform
              terraform-plugin-docs
              tflint
              actionlint
              just
              nixfmt
              git
              bash
              coreutils
              findutils
              python3
              gnupg
              gh
            ];

            # Keep Go and acceptance tests on the toolchain pinned by this flake.
            GOTOOLCHAIN = "local";
            TF_ACC_TERRAFORM_PATH = "${pkgs.terraform}/bin/terraform";
          };
        in
        {
          default = shell;
          terraform = shell;
        }
      );

      formatter = eachSystem (system: (pkgsFor system).nixfmt);
    };
}
