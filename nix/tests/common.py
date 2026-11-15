import json

import jwt


def setup_headscale(control):
    control.wait_for_unit("headscale.service")
    control.wait_for_open_port(80)
    control.wait_for_open_unix_socket("/run/headscale/headscale.sock")
    control.succeed('headscale users create test --display-name "NixOS Tests" --email "test@nixos.org"')

    users = json.loads(control.succeed("headscale users list --output json"))
    user_id = next(u["id"] for u in users if u["name"] == "test")

    authkey = json.loads(control.succeed(f"headscale preauthkeys create --user {user_id} --reusable --output json"))
    return authkey["key"]


def write_auth_key(m, key):
    with (m.shared_dir / "auth-key").open("w") as f:
        f.write(key)


def connect_tailscale(m, expected_ip=None, wait_connection=True):
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


def peer_ip(m, hostname):
    peers = json.loads(m.succeed("tailscale status --json")).get("Peer") or {}
    for peer in peers.values():
        if peer.get("HostName") == hostname:
            return next(ip for ip in peer["TailscaleIPs"] if "." in ip)
    return None


def fetch_token(client):
    client.wait_for_unit("tailfed-client.service")
    client.succeed("systemctl reload tailfed-client.service")
    client.wait_for_file("/run/tailfed/token", timeout=30)

    raw_token = client.succeed("cat /run/tailfed/token").strip()
    header = jwt.get_unverified_header(raw_token)
    claims = jwt.decode(raw_token, options={"verify_signature": False})
    print(header, claims)

    return raw_token, header, claims


def verify_signature(client, server_url, raw_token, header, claims):
    discovery = json.loads(client.succeed(f"curl -fsS {server_url}/.well-known/openid-configuration"))
    assert discovery.get("issuer") == claims.get("iss")

    jwks = client.succeed(f"curl -fsS {discovery['jwks_uri']}")
    keys = jwt.PyJWKSet.from_json(jwks)
    key = next(k for k in keys.keys if k.key_id == header.get("kid"))
    jwt.decode(raw_token, key=key, algorithms=[header["alg"]], audience=claims.get("aud"))


def verify_claims(server_url, header, claims, expected):
    assert header.get("alg") == expected["alg"]
    assert claims.get("iss") == server_url
    assert claims.get("aud") == expected["aud"]
    assert claims.get("sub") == expected["sub"]
    assert claims["exp"] - claims["iat"] == expected["lifetime"]
