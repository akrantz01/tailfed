# Show all the tasks
list:
    @just --list --unsorted

# Compile all the binaries
build: build-client build-dev

# Build only the client binary
build-client: (build-binary "client")

# Build only the dev binary
build-dev: (build-binary "dev")

[private]
build-binary binary:
    CGO_ENABLED=0 go build -o out/{{ binary }} ./cmd/{{ binary }}

# Run the mock backend server
mock port="8080":
    docker build -t tailfed-wiremock:latest wiremock/
    docker run --rm \
      --publish {{ port }}:8080/tcp \
      --volume $PWD/wiremock/stubs:/home/wiremock:ro \
      tailfed-wiremock:latest

# Remove the generated files
clean:
    rm -rf out

# Format everything
format:
    alejandra .
    go fmt ./...
    just --unstable --fmt
