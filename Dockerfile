FROM golang:1.24.1 AS build

ARG FUNCTION
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -tags lambda.norpc -o handler ./cmd/$FUNCTION

FROM public.ecr.aws/lambda/provided:al2023

COPY --from=docker.io/tailscale/tailscale:stable /usr/local/bin/tailscaled /var/runtime/tailscaled
COPY --from=docker.io/tailscale/tailscale:stable /usr/local/bin/tailscale /var/runtime/tailscale
RUN mkdir -p /var/run && ln -s /tmp/tailscale /var/run/tailscale && \
    mkdir -p /var/cache && ln -s /tmp/tailscale /var/cache/tailscale && \
    mkdir -p /var/lib && ln -s /tmp/tailscale /var/lib/tailscale && \
    mkdir -p /var/task && ln -s /tmp/tailscale /var/task/tailscale

COPY --chmod=755 ./scripts/docker-entrypoint.sh /var/runtime/bootstrap
COPY --from=build /app/handler ./handler

ENTRYPOINT [ "/var/runtime/bootstrap" ]
