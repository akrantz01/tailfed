{
  module,
  pkgs,
}: let
  nixOSVersion = "23.11";

  mock-api = pkgs.callPackage ./mock-api.nix {};
  tailscale-control = pkgs.buildGoModule {
    pname = "testcontrol";
    version = "1.82.5";

    src = pkgs.fetchFromGitHub {
      owner = "tailscale";
      repo = "tailscale";
      tag = "v1.82.5";
      hash = "sha256-BFitj8A+TfNKTyXBB1YhsEs5NvLUfgJ2IbjB2ipf4xU=";
    };
    vendorHash = "sha256-SiUkN6BQK1IQmLfkfPetzvYqRu9ENK6+6txtGxegF5Y=";

    subPackages = ["cmd/testcontrol"];
  };
in {
  module = pkgs.testers.runNixOSTest {
    name = "module";
    nodes = {
      api = {
        systemd.services.mock-api = {
          wants = ["network-online.target"];
          wantedBy = ["multi-user.target"];

          serviceConfig = {
            Type = "simple";
            ExecStart = "${mock-api}/bin/mock-api";
            Restart = "on-failure";
          };
        };

        networking.firewall.allowedTCPPorts = [8080];

        system.stateVersion = nixOSVersion;
      };

      client = {
        imports = [module];

        environment.systemPackages = with pkgs; [curl];

        services.tailscale = {
          enable = true;
          authKeyFile = pkgs.writers.writeText "test-auth-key" "dummy-auth-key";
          extraUpFlags = ["--login-server=http://127.0.0.1:9911"];
        };

        services.tailfed = {
          enable = true;
          url = "http://api:8080";
        };

        systemd.services.tailscale-control = {
          wants = ["network-online.target"];
          wantedBy = ["multi-user.target"];

          serviceConfig = {
            Type = "simple";
            ExecStart = "${tailscale-control}/bin/testcontrol";
          };
        };

        system.stateVersion = nixOSVersion;
      };
    };

    extraPythonPackages = p: [p.pyjwt];
    testScript = ''
      import jwt

      api.wait_for_unit("mock-api.service")
      api.wait_for_open_port(8080)
      api.succeed("curl -qv http://[::1]:8080/__admin/health", timeout=10)

      client.wait_for_unit("default.target")
      client.succeed("curl -qv http://api:8080/__admin/health", timeout=10)

      client.wait_for_unit("tailfed.service")
      client.wait_for_file("/var/lib/tailfed/token", timeout=10)
      client.copy_from_vm("/var/lib/tailfed/token")

      with (client.out_dir / "token").open("r") as f:
        token = f.read()
      claims = jwt.decode(token, options={"verify_signature": False})

      assert claims.get("iss") == "wiremock"
      assert claims.get("aud") == "wiremock.io"
      assert claims.get("sub") == "user-123"
    '';
  };
}
