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
      lib = pkgs.lib;

      version = "1.1.0";

      tailfed = pkgs.buildGoModule {
        pname = "tailfed";
        inherit version;

        src = lib.sources.sourceByRegex ./. ["^cmd$" "^cmd/.*" "^internal$" "^internal/.*" "^go\.(mod|sum)$"];
        vendorHash = "sha256-vUrZEApfPWEoijJCEHsHJeAUNiUpV25A2VtRbR2icCs=";

        subPackages = ["cmd/client"];
        env.CGO_ENABLED = 0;
      };
    in {
      packages = {
        inherit tailfed;
        default = tailfed;
      };

      nixosModules = let
        module = import ./nix/module.nix {inherit tailfed;};
      in {
        tailfed = module;
        default = module;
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
