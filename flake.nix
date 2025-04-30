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

      buildCmd = pname: dir: pkgs.buildGoModule {
        inherit pname;
        version = "1.1.0";
        src = lib.sources.sourceByRegex ./. ["^cmd$" "^cmd/.*" "^internal$" "^internal/.*" "^go\.(mod|sum)$"];
        vendorHash = "sha256-45snGcxMjoyAS4xYf89BtDHWQLj6U8hubuyzDLWd+6I=";

        subPackages = ["cmd/${dir}"];
        env.CGO_ENABLED = 0;
      };

      tailfed = buildCmd "tailfed" "client";
      dev = buildCmd "tailfed-dev" "dev";
      tscontrol = buildCmd "tailfed-tscontrol" "tscontrol";
    in {
      packages = {
        inherit tailfed dev tscontrol;
        default = tailfed;

        mock-api = pkgs.callPackage ./nix/mock-api.nix {};
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

      checks = import ./nix/tests.nix {
        inherit pkgs;
        module = self.nixosModules.${system}.default;
      };
    });
}
