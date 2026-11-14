{
  module,
  pkgs,
}: let
  lib = pkgs.lib;
  nixOSVersion = "23.11";

  # VLAN addresses are assigned alphabetically by node name, and tailnet addresses in the order the
  # test script connects the nodes
  controlAddress = "192.168.1.2";
  serverTailscaleAddress = "100.64.0.2";
  clientTailscaleAddress = "100.64.0.3";

  tailscaleNode = loginServer: {
    services.tailscale = {
      enable = true;
      authKeyFile = "/tmp/shared/auth-key";
      extraUpFlags = ["--login-server=${loginServer}"];
    };
    systemd.services = {
      tailscaled.environment.TS_DEBUG_USE_DERP_HTTP = "true";
      tailscaled-autoconnect.wantedBy = lib.mkForce [];
    };

    networking.firewall.enable = false;
    system.stateVersion = nixOSVersion;
  };

  moduleTest = {
    name,
    client ? {},
    assertions ? "",
  }:
    pkgs.testers.runNixOSTest {
      inherit name;

      nodes = {
        control = lib.mkMerge [
          (tailscaleNode "http://localhost")
          {
            services.headscale = {
              enable = true;
              address = "0.0.0.0";
              port = 80;

              settings = {
                server_url = "http://${controlAddress}";
                grpc_allow_insecure = true;

                log.level = "trace";

                prefixes = {
                  v4 = "100.64.0.0/10";
                  v6 = "fd7a:115c:a1e0::/48";
                  allocation = "sequential";
                };

                derp.server = {
                  enabled = true;
                  region_id = 1;
                  region_code = "headscale";
                  region_name = "Headscale Embedded DERP";

                  stun_listen_addr = "0.0.0.0:3478";

                  ipv4 = controlAddress;
                  ipv6 = "2001:db8:1::2";
                };
                derp.urls = [];

                dns = {
                  magic_dns = false;
                  override_local_dns = false;
                };
              };
            };
            systemd.services.tailscaled = {
              requires = ["headscale.service"];
              after = ["headscale.service"];
              wantedBy = lib.mkForce [];
            };
          }
        ];

        server = lib.mkMerge [
          (tailscaleNode "http://${controlAddress}")
          {
            imports = [module];

            services.tailfed.server = {
              enable = true;
              settings = {
                log-level = "debug";
                listener.address = "0.0.0.0:8000";
                token.profiles = {
                  default.audience = "sts.amazonaws.com";
                  other = {
                    audience = "other.example.com";
                    accepted-algorithms = ["ES256"];
                    subject.user = ["hostname" "user.id"];
                    validity = "15m";
                  };
                };
              };
            };
          }
        ];

        client = lib.mkMerge [
          (tailscaleNode "http://${controlAddress}")
          {
            imports = [module];

            services.tailfed.client = {
              enable = true;
              settings = {
                log-level = "debug";
                url = "http://${serverTailscaleAddress}:8000";
              };
            };
          }
          client
        ];
      };

      extraPythonPackages = p: [p.pyjwt p.cryptography];
      testScript = ''
        import json
        import jwt

        control.wait_for_unit("headscale.service")
        control.wait_for_open_port(80)
        control.wait_for_open_unix_socket("/run/headscale/headscale.sock")
        control.succeed('headscale users create test --display-name "NixOS Tests" --email "test@nixos.org"')

        users = json.loads(control.succeed("headscale users list --output json"))
        user_id = next(u["id"] for u in users if u["name"] == "test")

        created_authkey = json.loads(control.succeed(f"headscale preauthkeys create --user {user_id} --reusable --output json"))
        for m in machines:
          with (m.shared_dir / "auth-key").open("w") as f:
            f.write(created_authkey["key"])

        control.start_job("tailscaled.service")

        def connect_tailscale(m, expected_ip = None, wait_connection = True):
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
          m.succeed("systemctl start tailscaled-autoconnect.service")

          with m.nested("waiting for tailscale to be running"):
            retry(tailscale_running, timeout_seconds=30)

          if expected_ip is not None:
            ip = m.succeed("tailscale ip -4").strip()
            assert ip == expected_ip, f"expected tailscale address {expected_ip}, got {ip}"

          if wait_connection:
            with m.nested("waiting for tailscale connection"):
              retry(tailscale_connected, timeout_seconds=30)

        connect_tailscale(control, wait_connection=False)
        connect_tailscale(server, expected_ip="${serverTailscaleAddress}")
        connect_tailscale(client, expected_ip="${clientTailscaleAddress}")

        server.wait_for_unit("tailfed-server.service")
        server.wait_for_open_port(8000)
        server.succeed("curl -fsS http://127.0.0.1:8000/health", timeout=10)

        with subtest("token is not issued outside the tailnet"):
          status = control.succeed("curl -s -o /dev/null -w '%{http_code}' -H 'Sec-Tailfed: 1' http://server:8000/token", timeout=10)
          assert status == "401", f"expected 401, got {status}"

        with subtest("client fetches a token"):
          client.wait_for_unit("tailfed-client.service")
          client.succeed("systemctl reload tailfed-client.service")
          client.wait_for_file("/run/tailfed/token", timeout=30)
          raw_token = client.succeed("cat /run/tailfed/token").strip()
          header = jwt.get_unverified_header(raw_token)
          claims = jwt.decode(raw_token, options={"verify_signature": False})
          print(header, claims)

        with subtest("token signature matches the published keys"):
          discovery = json.loads(client.succeed("curl -fsS http://${serverTailscaleAddress}:8000/.well-known/openid-configuration"))
          assert discovery.get("issuer") == claims.get("iss")

          jwks = client.succeed(f"curl -fsS {discovery['jwks_uri']}")
          keys = jwt.PyJWKSet.from_json(jwks)
          key = next(k for k in keys.keys if k.key_id == header.get("kid"))
          jwt.decode(raw_token, key=key, algorithms=[header["alg"]], audience=claims.get("aud"))

        with subtest("token claims"):
          assert claims.get("iss") == "http://${serverTailscaleAddress}:8000"
          ${assertions}
      '';
    };
in {
  module-default-profile = moduleTest {
    name = "module-default-profile";
    assertions = ''
      assert claims.get("aud") == "sts.amazonaws.com"
      assert claims.get("sub") == "tailnet.dns::hostname:client:user.login:test@nixos.org"
      assert claims["exp"] - claims["iat"] == 60 * 60
    '';
  };

  module-custom-profile = moduleTest {
    name = "module-custom-profile";
    client.services.tailfed.client.settings.profile = "other";
    assertions = ''
      assert header.get("alg") == "ES256"
      assert claims.get("aud") == "other.example.com"
      assert claims.get("sub") == "hostname:client:user.id:1"
      assert claims["exp"] - claims["iat"] == 15 * 60
    '';
  };
}
