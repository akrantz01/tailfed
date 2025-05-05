{tailfed, ...}: {
  config,
  options,
  lib,
  pkgs,
  ...
}: let
  cfg = config.services.tailfed;
in {
  options.services.tailfed = {
    enable = lib.mkEnableOption "Enable the tailfed client daemon";

    package = lib.mkOption {
      type = lib.types.package;
      default = tailfed;
      defaultText = lib.literalExpression "tailfed";
      description = "The tailfed client package to use";
    };

    logLevel = lib.mkOption {
      type = lib.types.enum ["panic" "fatal" "error" "warn" "info" "debug" "trace"];
      default = "info";
      description = "The minimum level to emit logs at";
    };

    tokenPath = lib.mkOption {
      type = lib.types.str;
      default = "/run/tailfed/token";
      description = "Where to write the created token";
    };

    url = lib.mkOption {
      type = lib.types.str;
      example = "https://tailfed.example.com";
      description = "The tailfed API URL to connect to";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.tailfed = {
      description = "tailfed client daemon";
      after = ["network-online.target" "tailscaled.service"];
      wants = ["network-online.target"];
      requires = ["tailscaled.service"];
      wantedBy = ["multi-user.target"];

      restartTriggers = [cfg.package];

      environment = {
        TAILFED_LOG_LEVEL = cfg.logLevel;
        TAILFED_PID_FILE = "%S/tailfed/pid";
        TAILFED_PATH = cfg.tokenPath;
        TAILFED_URL = cfg.url;
      };

      serviceConfig = {
        Type = "notify";
        WatchdogSec = 30;

        User = "tailfed";
        DynamicUser = "yes";
        SyslogIdentifier = "tailfed";
        StateDirectory = "tailfed";
        RuntimeDirectory = "tailfed";
        RuntimeDirectoryMode = "0755";

        ExecStart = "${cfg.package}/bin/client";
        ExecReload = "${cfg.package}/bin/client refresh --wait";
        Restart = "on-failure";
      };
    };
  };
}
