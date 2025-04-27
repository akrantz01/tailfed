# Show all the tasks
list:
    @just --list --unsorted

# Compile all the binaries
build: build-client build-dev build-lambdas

# Build only the client binary
build-client: (build-binary "client")

# Build only the dev binary
build-dev: (build-binary "dev")

# Build only the Lambda binaries
build-lambdas: (build-binary "finalizer") (build-binary "generator") (build-binary "initializer") (build-binary "verifier")

# Build and package all the Lambda functions
package-lambdas: (package-lambda "finalizer") (package-lambda "generator") (package-lambda "initializer") (package-lambda "verifier")

[private]
package-lambda binary: (build-binary binary)
    cp out/{{ binary }} out/bootstrap
    zip --junk-paths out/{{ binary }}.zip out/bootstrap
    rm out/bootstrap

[private]
build-binary binary:
    CGO_ENABLED=0 go build -o out/{{ binary }} ./cmd/{{ binary }}

# Run the mock backend server
mock port="8080":
    docker build -t tailfed-wiremock:latest wiremock/
    docker run --rm \
      --publish mode=host,target={{ port }},published=8080,protocol=tcp \
      --volume type=bind,source=$PWD/wiremock/stubs,target=/home/wiremock \
      tailfed-wiremock:latest

# Remove the generated files
clean:
    rm -rf out

# Format everything
format:
    alejandra .
    go fmt ./...
    just --unstable --fmt
