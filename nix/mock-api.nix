{
  fetchurl,
  jre,
  lib,
  makeWrapper,
  stdenvNoCC,
  stubsPath ? ../wiremock/stubs,
}: let
  versions = {
    wiremock = "3.12.1";
    wiremock-jwt = "0.2.0";
  };

  wiremock = fetchurl {
    url = "mirror://maven/org/wiremock/wiremock-standalone/${versions.wiremock}/wiremock-standalone-${versions.wiremock}.jar";
    hash = "sha256-OoyH8wqvMQvLHPi4bTRodINMpqzZAdsvixm5qmqAZuI=";
  };
  wiremock-jwt = fetchurl {
    url = "mirror://maven/org/wiremock/extensions/wiremock-jwt-extension-standalone/${versions.wiremock-jwt}/wiremock-jwt-extension-standalone-${versions.wiremock-jwt}.jar";
    hash = "sha256-JY4841YrrFJOGXI6rY2EIbWlW5mAdY9bjfvelmPqgWA=";
  };

  stubs = lib.sources.cleanSource stubsPath;
in
  stdenvNoCC.mkDerivation (finalAttrs: {
    pname = "mock-api";
    version = "1.1.0";

    src = null;
    nativeBuildInputs = [makeWrapper jre];

    phases = ["installPhase" "fixupPhase"];

    installPhase = ''
      mkdir -p "$out"/{share/wiremock/{lib,extensions},etc/wiremock,bin}
      cp ${wiremock} "$out/share/wiremock/lib/wiremock.jar"
      cp ${wiremock-jwt} "$out/share/wiremock/extensions/wiremock-jwt-extension.jar"
      cp -r "${stubs}"/* "$out/etc/wiremock/"

      makeWrapper ${jre}/bin/java $out/bin/mock-api \
        --add-flags "-cp $out/share/wiremock/lib/*:$out/share/wiremock/extensions/*" \
        --add-flags "wiremock.Run" \
        --add-flags "--root-dir $out/etc/wiremock" \
        --add-flags "--global-response-templating"
    '';

    meta = {
      description = "Stubbed API for tailfed testing";
      mainProgram = "mock-api";
      platforms = jre.meta.platforms;
    };
  })
