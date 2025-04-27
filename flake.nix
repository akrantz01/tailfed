{
  description = "Turn your Tailscale network into an AWS web identity federation-compatible OpenID Connect provider";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};

      version = "1.1.0";

      tailfed = pkgs.buildGoModule {
        pname = "tailfed";
        inherit version;

        src = ./.;
        vendorHash = "sha256-vUrZEApfPWEoijJCEHsHJeAUNiUpV25A2VtRbR2icCs=";

        subPackages = ["cmd/client"];
        env.CGO_ENABLED = 0;
      };
    in {
      packages = {
        inherit tailfed;
        default = tailfed;
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [go gopls gotools go-tools];
        packages = with pkgs; [alejandra just];

        shellHook = ''
          alias j=just
        '';
      };
    });
}
