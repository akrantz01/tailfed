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

      rawRev = self.rev or self.dirtyRev or "unknown";
      rev = lib.strings.removeSuffix "-dirty" rawRev;
      dirty = (builtins.stringLength rawRev) != (builtins.stringLength rev);

      buildCmd = pname: dir:
        pkgs.buildGoModule rec {
          inherit pname;
          version = "1.1.0";
          src = lib.sources.sourceByRegex ./. ["^cmd$" "^cmd/.*" "^internal$" "^internal/.*" "^go\.(mod|sum)$"];
          vendorHash = "sha256-pxdpHTSNVuY9HiB5M2ZNt+ymzBTikTmjpKSQ3MFXn34=";

          subPackages = ["cmd/${dir}"];
          env.CGO_ENABLED = 0;

          ldflags = [
            "-X github.com/akrantz01/tailfed/internal/version.version=${version}"
            "-X github.com/akrantz01/tailfed/internal/version.commit=${rev}"
            "-X github.com/akrantz01/tailfed/internal/version.dirty=${lib.trivial.boolToString dirty}"
            "-X github.com/akrantz01/tailfed/internal/version.date=${builtins.toString self.lastModified}"
          ];

          meta = {
            description = "Turn your Tailscale network into an AWS web identity federation-compatible OpenID Connect provider";
            homepage = "https://github.com/akrantz01/tailfed";
            changelog = "https://github.com/akrantz01/tailfed/blob/v${version}/CHANGELOG.md";
            releases = "https://github.com/akrantz01/tailfed/releases/latest";
            mainProgram = dir;
            license = lib.licenses.mit;
            platforms = lib.platforms.all;
          };
        };

      tailfed = buildCmd "tailfed" "client";
      dev = buildCmd "tailfed-dev" "dev";
    in {
      packages = {
        inherit tailfed dev;
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
        inherit pkgs dev;
        module = self.nixosModules.${system}.default;
      };
    });
}
