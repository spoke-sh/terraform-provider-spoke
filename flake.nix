{
  description = "Development environment for the Spoke Terraform provider";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/1e8bc658fc985ef27ccd66d107d767b32bb7ef98";

  outputs =
    { nixpkgs, ... }:
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
