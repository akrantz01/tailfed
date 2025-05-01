{
  module,
  pkgs,
  dev,
}: let
  lib = pkgs.lib;
  nixOSVersion = "23.11";

  mock-api = pkgs.callPackage ./mock-api.nix {};

  moduleTest = {
    name,
    preScript ? "",
    assertions ? "",
    api ? {},
    client ? {},
  }:
    pkgs.testers.runNixOSTest {
      inherit name;

      nodes = {
        control = {
          services.headscale = {
            enable = true;
            address = "0.0.0.0";
            port = 80;

            settings = {
              server_url = "http://192.168.1.3";
              grpc_allow_insecure = true;

              log.level = "trace";

              prefixes = {
                v4 = "100.64.0.0/10";
                v6 = "fd7a:115c:a1e0::/48";
              };

              derp.server = {
                enabled = true;
                region_id = 1;
                region_code = "headscale";
                region_name = "Headscale Embedded DERP";

                stun_listen_addr = "0.0.0.0:3478";

                # IP addresses are allocated alphabetically based on hostname
                ipv4 = "192.168.1.3";
                ipv6 = "2001:db8:1::3";
              };
              derp.urls = [];

              dns.magic_dns = false;
            };
          };
          services.tailscale = {
            enable = true;
            authKeyFile = "/tmp/shared/auth-key";
            extraUpFlags = ["--login-server=http://localhost"];
          };
          systemd.services = {
            tailscaled = {
              requires = ["headscale.service"];
              after = ["headscale.service"];
              wantedBy = lib.mkForce [];
              environment.TS_DEBUG_USE_DERP_HTTP = "true";
            };
            tailscaled-autoconnect.wantedBy = lib.mkForce [];
          };

          networking.firewall.enable = false;
          system.stateVersion = nixOSVersion;
        };

        api =
          lib.attrsets.recursiveUpdate {
            services.tailscale = {
              enable = true;
              authKeyFile = "/tmp/shared/auth-key";
              extraUpFlags = ["--login-server=http://192.168.1.3"];
            };
            systemd.services.tailscaled.environment.TS_DEBUG_USE_DERP_HTTP = "true";

            networking.firewall.enable = false;
            system.stateVersion = nixOSVersion;
          }
          api;

        client =
          lib.attrsets.recursiveUpdate {
            imports = [module];

            services.tailscale = {
              enable = true;
              authKeyFile = "/tmp/shared/auth-key";
              extraUpFlags = ["--login-server=http://192.168.1.3"];
            };
            systemd.services.tailscaled.environment.TS_DEBUG_USE_DERP_HTTP = "true";

            services.tailfed = {
              enable = true;
              logLevel = "debug";
              url = "http://192.168.1.1:8080";
            };

            networking.firewall.enable = false;
            system.stateVersion = nixOSVersion;
          }
          client;
      };

      extraPythonPackages = p: [p.pyjwt];
      testScript = ''
        import json
        import jwt

        control.wait_for_unit("headscale.service")
        control.wait_for_open_port(80)
        control.wait_for_open_unix_socket("/run/headscale/headscale.sock")
        control.succeed('headscale users create test --display-name "NixOS Tests" --email "test@nixos.org"')

        created_authkey = json.loads(control.succeed("headscale preauthkeys create --user test --reusable --output json"))
        for m in machines:
          with (m.shared_dir / "auth-key").open("w") as f:
            f.write(created_authkey["key"])

        apikey = json.loads(control.succeed("headscale apikeys create --output json"))
        with (api.shared_dir / "api-key").open("w") as f:
          f.write(apikey)

        control.start_job("tailscaled.service")
        control.start_job("tailscaled-autoconnect.service")

        def wait_for_tailscale(m, wait_connection = True):
          def tailscale_running(_last_try):
            exit_code, output = m.execute("tailscale status --json --peers=false", timeout=10)
            if exit_code != 0:
              return False

            return json.loads(output).get("BackendState") == "Running"

          def tailscale_connected(_last_try):
            exit_code, output = m.execute("ping -c 1 -W 1 100.64.0.1", timeout=10)
            print(output)
            return exit_code == 0

          m.wait_for_unit("tailscaled.service")

          with m.nested("waiting for tailscale to be running"):
            retry(tailscale_running, timeout=30)

          if wait_connection:
            with m.nested("waiting for tailscale connection"):
              retry(tailscale_connected, timeout=30)

        wait_for_tailscale(control, wait_connection=False)

        for m in api, client:
          wait_for_tailscale(m)

        ${preScript}

        client.wait_for_unit("tailfed.service")
        client.wait_for_file("/var/lib/tailfed/token", timeout=10)
        client.copy_from_vm("/var/lib/tailfed/token")

        with (client.out_dir / "token").open("r") as f:
          token = f.read()
        claims = jwt.decode(token, options={"verify_signature": False})
        ${assertions}
      '';
    };
in {
  module-mock = moduleTest {
    name = "module-mock";
    api.systemd.services.mock-api = {
      wants = ["network-online.target"];
      wantedBy = ["multi-user.target"];

      serviceConfig = {
        Type = "simple";
        ExecStart = "${mock-api}/bin/mock-api";
        Restart = "on-failure";
      };
    };

    preScript = ''
      api.wait_for_unit("mock-api.service")
      api.wait_for_open_port(8080)
      api.succeed("curl -qv http://[::1]:8080/__admin/health", timeout=10)
    '';
    assertions = ''
      assert claims.get("iss") == "wiremock"
      assert claims.get("aud") == "wiremock.io"
      assert claims.get("sub") == "user-123"
    '';
  };

  module-dev = moduleTest {
    name = "module-dev";
    api.systemd.services.dev-api = {
      wants = ["network-online.target"];
      wantedBy = ["multi-user.target"];

      environment = {
        DEV_GATEWAY_LOG_LEVEL = "debug";
        DEV_GATEWAY_ADDRESS = "0.0.0.0:8080";
        DEV_GATEWAY_TAILSCALE__BACKEND = "headscale";
        DEV_GATEWAY_TAILSCALE__BASE_URL = "192.168.1.3:50443";
        DEV_GATEWAY_TAILSCALE__TAILNET = "192.168.1.3";
        DEV_GATEWAY_TAILSCALE__API_KEY = "file:///tmp/shared/api-key";
        DEV_GATEWAY_TAILSCALE__TLS_MODE = "none";
      };

      serviceConfig = {
        Type = "simple";
        ExecStart = "${dev}/bin/dev";
        Restart = "on-failure";
      };
    };

    preScript = ''
      api.wait_for_unit("dev-api.service")
      api.wait_for_open_port(8080)
      api.succeed("curl -qv http://[::1]:8080/health", timeout=10)
    '';
    assertions = ''
      assert claims.get("iss") == "https://mock-api.execute-api.us-east-1.amazonaws.com"
      assert claims.get("aud") == "sts.amazonaws.com"
      assert claims.get("sub") == "3"
      assert claims.get("tailnet") == "192.168.1.3"
      assert claims.get("authorized") is True
      assert claims.get("external") is False
      assert claims.get("os") == "unknown"
      assert claims.get("dns_name") == "client"
      assert claims.get("host_name") == "client"
      assert claims.get("machine_name") == "client"
      assert claims.get("tags") == []
    '';
  };
}
