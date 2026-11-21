{
  lib,
  stdenvNoCC,
  fetchurl,
  installShellFiles,
}:
let
  version = "2.20.2";

  sources = {
    x86_64-linux = {
      platform = "Linux_x86_64";
      hash = "sha256-tBeOfJNuwVeaNgVqekEtJEvWU/evH7JBPyyGYrmcdvs=";
    };
    aarch64-linux = {
      platform = "Linux_arm64";
      hash = "sha256-3v6OMH7AO7ZOoklU5Oz4Qh9kifV7EvrDgC5fRisDHpM=";
    };
    x86_64-darwin = {
      platform = "Darwin_x86_64";
      hash = "sha256-VcRnLWH8NJzKA9RmupcS7cvgzCia0B78PlLUEkKStBI=";
    };
    aarch64-darwin = {
      platform = "Darwin_arm64";
      hash = "sha256-1JQ5QZpECVwA30LLHAzOaY9SOfHz5DDturaJboo/UCE=";
    };
  };

  source =
    sources.${stdenvNoCC.hostPlatform.system}
      or (throw "goreleaser-pro: unsupported system ${stdenvNoCC.hostPlatform.system}");
in
stdenvNoCC.mkDerivation {
  pname = "goreleaser-pro";
  inherit version;

  src = fetchurl {
    url = "https://github.com/goreleaser/goreleaser-pro/releases/download/v${version}/goreleaser-pro_${source.platform}.tar.gz";
    inherit (source) hash;
  };

  sourceRoot = ".";

  nativeBuildInputs = [ installShellFiles ];

  installPhase = ''
    runHook preInstall

    install -Dm755 goreleaser "$out/bin/goreleaser"
    installManPage manpages/goreleaser.1.gz
    installShellCompletion \
      --bash completions/goreleaser.bash \
      --fish completions/goreleaser.fish \
      --zsh completions/goreleaser.zsh

    runHook postInstall
  '';

  meta = {
    description = "Release automation tool for Go projects (Pro edition)";
    homepage = "https://goreleaser.com/pro/";
    license = lib.licenses.unfree;
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
    mainProgram = "goreleaser";
    platforms = builtins.attrNames sources;
  };
}
