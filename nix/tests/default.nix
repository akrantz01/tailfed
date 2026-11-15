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

  controlNode = lib.mkMerge [
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

  serverSettings = {
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

  clientNode = extra:
    lib.mkMerge [
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
      extra
    ];

  script = file: expected:
    ''
      SERVER_IP = "${serverTailscaleAddress}"
      CLIENT_IP = "${clientTailscaleAddress}"
      EXPECTED = ${builtins.toJSON expected}
    ''
    + builtins.readFile ./common.py
    + builtins.readFile file;

  localTest = {
    name,
    client ? {},
    expected,
  }:
    pkgs.testers.runNixOSTest {
      inherit name;

      nodes = {
        control = controlNode;

        server = lib.mkMerge [
          (tailscaleNode "http://${controlAddress}")
          {
            imports = [module];

            services.tailfed.server = {
              enable = true;
              settings = serverSettings;
            };
          }
        ];

        client = clientNode client;
      };

      extraPythonPackages = p: [p.pyjwt p.cryptography];
      testScript = script ./local.py expected;
    };
in {
  module-default-profile = localTest {
    name = "module-default-profile";
    expected = {
      alg = "RS256";
      aud = "sts.amazonaws.com";
      sub = "tailnet.dns::hostname:client:user.login:test@nixos.org";
      lifetime = 60 * 60;
    };
  };

  module-custom-profile = localTest {
    name = "module-custom-profile";
    client.services.tailfed.client.settings.profile = "other";
    expected = {
      alg = "ES256";
      aud = "other.example.com";
      sub = "hostname:client:user.id:1";
      lifetime = 15 * 60;
    };
  };

  module-in-process = pkgs.testers.runNixOSTest {
    name = "module-in-process";

    nodes = {
      control = controlNode;

      server = {
        imports = [module];

        services.tailfed.server = {
          enable = true;
          environmentFile = "/tmp/shared/server.env";
          settings =
            serverSettings
            // {
              tailscale = {
                connection = "in-process";
                control-plane-url = "http://${controlAddress}";
              };
            };
        };
        # Without root, tsnet binds its sockets to the default route's interface
        networking.defaultGateway = {
          address = controlAddress;
          interface = "eth1";
        };

        systemd.services.tailfed-server = {
          wantedBy = lib.mkForce [];
          environment.TS_DEBUG_USE_DERP_HTTP = "true";
        };

        networking.firewall.enable = false;
        system.stateVersion = nixOSVersion;
      };

      client = clientNode {};
    };

    extraPythonPackages = p: [p.pyjwt p.cryptography];
    testScript = script ./in_process.py {
      alg = "RS256";
      aud = "sts.amazonaws.com";
      sub = "tailnet.dns::hostname:client:user.login:test@nixos.org";
      lifetime = 60 * 60;
    };
  };
}
