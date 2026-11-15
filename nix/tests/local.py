server_url = f"http://{SERVER_IP}:8000"

authkey = setup_headscale(control)
for m in machines:
    write_auth_key(m, authkey)

control.start_job("tailscaled.service")

connect_tailscale(control, wait_connection=False)
connect_tailscale(server, expected_ip=SERVER_IP)
connect_tailscale(client, expected_ip=CLIENT_IP)

server.wait_for_unit("tailfed-server.service")
server.wait_for_open_port(8000)
server.succeed("curl -fsS http://127.0.0.1:8000/health", timeout=10)

with subtest("token is not issued outside the tailnet"):
    status = control.succeed(
        "curl -s -o /dev/null -w '%{http_code}' -H 'Sec-Tailfed: 1' http://server:8000/token",
        timeout=10,
    )
    assert status == "401", f"expected 401, got {status}"

with subtest("client fetches a token"):
    raw_token, header, claims = fetch_token(client)

with subtest("token signature matches the published keys"):
    verify_signature(client, server_url, raw_token, header, claims)

with subtest("token claims"):
    verify_claims(f"http://{peer_dns_name(client, 'server')}:8000", header, claims, EXPECTED)
