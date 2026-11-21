#!/usr/bin/env bash
set -euo pipefail

METADATA="http://metadata.google.internal/computeMetadata/v1"

attribute() {
  curl -fsS --retry 5 --retry-connrefused -H "Metadata-Flavor: Google" "${METADATA}/instance/attributes/$1"
}

case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

artifacts_url="$(attribute tailfed-artifacts-url)"
archive="tailfed-server_linux_${arch}.tar.gz"

workdir="$(mktemp -d)"
trap 'rm -rf "${workdir}"' EXIT

curl -fsS --retry 5 -o "${workdir}/checksums.txt" "${artifacts_url}/checksums.txt"
curl -fsS --retry 5 -o "${workdir}/${archive}" "${artifacts_url}/${archive}"

(cd "${workdir}" && grep -E "^[0-9a-f]{64}  ${archive}\$" checksums.txt | sha256sum --check --strict)

tar -xzf "${workdir}/${archive}" -C "${workdir}" tailfed-server
install -m 0755 "${workdir}/tailfed-server" /usr/local/bin/tailfed-server

install -d -m 0755 /etc/tailfed-server
attribute tailfed-config > /etc/tailfed-server/config.yml.new
chmod 0644 /etc/tailfed-server/config.yml.new
mv /etc/tailfed-server/config.yml.new /etc/tailfed-server/config.yml

cat > /etc/systemd/system/tailfed-server.service << 'EOF'
[Unit]
Description=tailfed-server OpenID Connect provider
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/tailfed-server --config /etc/tailfed-server/config.yml --json
Restart=on-failure
RestartSec=5

User=tailfed-server
DynamicUser=true
StateDirectory=tailfed-server

NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK
SystemCallArchitectures=native

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tailfed-server.service
systemctl restart tailfed-server.service
