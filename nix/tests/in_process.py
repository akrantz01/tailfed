server_url = f"http://{SERVER_IP}:8000"

authkey = setup_headscale(control)
for m in control, client:
    write_auth_key(m, authkey)
with (server.shared_dir / "server.env").open("w") as f:
    f.write(f"TAILFED_SERVER_TAILSCALE__AUTH_KEY={authkey}\n")

control.start_job("tailscaled.service")

connect_tailscale(control, wait_connection=False)

server.succeed("systemctl start tailfed-server.service")
with server.nested("waiting for tailfed-server to join the tailnet"):
    retry(lambda _: peer_ip(control, "server") is not None, timeout_seconds=60)
server_ip = peer_ip(control, "server")
assert server_ip == SERVER_IP, f"expected tailscale address {SERVER_IP}, got {server_ip}"

connect_tailscale(client, expected_ip=CLIENT_IP)

client.wait_until_succeeds(f"curl -fsS {server_url}/health", timeout=60)

with subtest("server is not reachable outside the tailnet"):
    control.fail("curl -fsS --max-time 5 http://server:8000/health")

with subtest("client fetches a token"):
    raw_token, header, claims = fetch_token(client)

with subtest("token signature matches the published keys"):
    verify_signature(client, server_url, raw_token, header, claims)

with subtest("token claims"):
    verify_claims(f"http://{peer_dns_name(client, 'server')}:8000", header, claims, EXPECTED)
