{
  client,
  server,
  ...
}: {
  config,
  lib,
  pkgs,
  ...
}: let
  cfg = config.services.tailfed;
  yaml = pkgs.formats.yaml {};

  logLevel = lib.mkOption {
    type = lib.types.enum ["panic" "fatal" "error" "warn" "info" "debug" "trace"];
    default = "info";
    description = "The minimum level to emit logs at";
  };

  tailscaleUnits = lib.optionals config.services.tailscale.enable ["tailscaled.service"];
in {
  options.services.tailfed = {
    client = {
      enable = lib.mkEnableOption "the tailfed-client daemon";

      package = lib.mkOption {
        type = lib.types.package;
        default = client;
        defaultText = lib.literalExpression "tailfed-client";
        description = "The tailfed-client package to use";
      };

      settings = lib.mkOption {
        description = ''
          Configuration for tailfed-client, rendered to YAML. See `tailfed.example.yml` for all
          available options.
        '';
        default = {};
        type = lib.types.submodule {
          freeformType = yaml.type;

          options = {
            log-level = logLevel;

            path = lib.mkOption {
              type = lib.types.str;
              default = "/run/tailfed/token";
              description = "Where to write the issued token";
            };

            profile = lib.mkOption {
              type = lib.types.str;
              default = "default";
              description = "The server-side profile to request tokens for";
            };

            url = lib.mkOption {
              type = lib.types.str;
              example = "https://tailfed.example.com";
              description = "The tailfed-server URL to connect to";
            };
          };
        };
      };
    };

    server = {
      enable = lib.mkEnableOption "the tailfed-server OpenID Connect provider";

      package = lib.mkOption {
        type = lib.types.package;
        default = server;
        defaultText = lib.literalExpression "tailfed-server";
        description = "The tailfed-server package to use";
      };

      environmentFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        example = "/run/secrets/tailfed-server.env";
        description = ''
          An optional file of `TAILFED_SERVER_*` environment variables, for values that should not
          be placed in the world-readable Nix store (e.g. `TAILFED_SERVER_TAILSCALE__AUTH_KEY`).
          Values may be `file://` paths or AWS secret ARNs.
        '';
      };

      settings = lib.mkOption {
        description = ''
          Configuration for tailfed-server, rendered to YAML. See `tailfed-server.example.yml` for
          all available options.
        '';
        default = {};
        example = lib.literalExpression ''
          {
            listener.http = "0.0.0.0:8000";
            token.profiles.default.audience = "sts.amazonaws.com";
          }
        '';
        type = lib.types.submodule {
          freeformType = yaml.type;

          options = {
            log-level = logLevel;

            tailscale = lib.mkOption {
              default = {};
              type = lib.types.submodule {
                freeformType = yaml.type;

                options.connection = lib.mkOption {
                  type = lib.types.enum ["local" "in-process"];
                  default = "local";
                  description = "How to connect to Tailscale";
                };
              };
            };
          };
        };
      };
    };
  };

  config = lib.mkMerge [
    (lib.mkIf cfg.client.enable {
      systemd.services.tailfed-client = {
        description = "tailfed-client daemon";
        after = ["network-online.target"] ++ tailscaleUnits;
        wants = ["network-online.target"];
        requires = tailscaleUnits;
        wantedBy = ["multi-user.target"];

        restartTriggers = [cfg.client.package];

        serviceConfig = {
          Type = "notify";
          WatchdogSec = 30;

          User = "tailfed";
          DynamicUser = true;
          SyslogIdentifier = "tailfed-client";
          RuntimeDirectory = "tailfed";
          RuntimeDirectoryMode = "0755";

          ExecStart = "${lib.getExe cfg.client.package} --config ${yaml.generate "tailfed.yml" cfg.client.settings}";
          ExecReload = "${pkgs.coreutils}/bin/kill -HUP $MAINPID";
          Restart = "on-failure";
        };
      };
    })

    (lib.mkIf cfg.server.enable {
      systemd.services.tailfed-server = let
        local = cfg.server.settings.tailscale.connection == "local";
        units = lib.optionals local tailscaleUnits;
      in {
        description = "tailfed-server OpenID Connect provider";
        after = ["network-online.target"] ++ units;
        wants = ["network-online.target"];
        requires = units;
        wantedBy = ["multi-user.target"];

        restartTriggers = [cfg.server.package];

        environment = lib.mkIf (!local) {
          XDG_CONFIG_HOME = "%S/tailfed-server";
        };

        serviceConfig = {
          Type = "simple";

          User = "tailfed-server";
          DynamicUser = true;
          SyslogIdentifier = "tailfed-server";
          StateDirectory = "tailfed-server";
          EnvironmentFile = lib.mkIf (cfg.server.environmentFile != null) cfg.server.environmentFile;

          ExecStart = "${lib.getExe cfg.server.package} --config ${yaml.generate "tailfed-server.yml" cfg.server.settings}";
          Restart = "on-failure";
        };
      };
    })
  ];
}
