#!/bin/sh

mkdir -p /tmp/tailscale
/var/runtime/tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
/var/runtime/tailscale up \
  --auth-key="${TAILSCALE_AUTHKEY}" \
  --hostname="${AWS_LAMBDA_FUNCTION_NAME}-${AWS_LAMBDA_FUNCTION_VERSION}-${HOSTNAME}"
ALL_PROXY=socks5://localhost:1055/ /var/task/handler
