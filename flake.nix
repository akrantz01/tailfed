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
          version = "2.0.0";
          src = lib.sources.sourceByRegex ./. ["^cmd$" "^cmd/.*" "^internal$" "^internal/.*" "^go\.(mod|sum)$"];
          vendorHash = "sha256-z2KqrzABerPH90PVV6ayq4ye02yui64CFDDtuRVhBVk=";

          subPackages = ["cmd/${dir}"];
          env.CGO_ENABLED = 0;

          postInstall = ''
            mv "$out/bin/${dir}" "$out/bin/${pname}"
          '';

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
            mainProgram = pname;
            license = lib.licenses.mit;
            platforms = lib.platforms.all;
          };
        };

      client = buildCmd "tailfed-client" "client";
      server = buildCmd "tailfed-server" "server";
    in {
      packages = {
        inherit client server;
      };

      nixosModules = let
        module = import ./nix/module.nix {inherit client server;};
      in {
        tailfed = module;
        default = module;
      };

      devShells.default = pkgs.mkShell {
        buildInputs = with pkgs; [go gopls gotools go-tools];
        packages = with pkgs; [alejandra];
      };

      checks = import ./nix/tests {
        inherit pkgs;
        module = self.nixosModules.${system}.default;
      };
    });
}
