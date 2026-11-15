GO_ARGS :=

# Directory for build artifacts
OUT_DIR := out

.PHONY: all build build-client build-server clean coverage help test

help:
	$(info help         - display this message)
	$(info build        - compile all the applications)
	$(info build-client - build only the client application)
	$(info build-server - build only the server application)
	$(info test         - run unit tests)
	$(info coverage     - run unit tests with coverage)
	$(info clean        - remove build artifacts)
	$(info all          - build everything for deployment)

all: build

build: build-client build-server

build-client: $(OUT_DIR)/client
build-server: $(OUT_DIR)/server

test:
	go test $(GO_ARGS) ./...

coverage: cover.out
	go tool cover -html cover.out

clean:
	rm -rf $(OUT_DIR) cover.out

# Create output directories
$(OUT_DIR) $(ZIP_DIR):
	mkdir -p $@

# Binary build rule
$(OUT_DIR)/%: | $(OUT_DIR)
	go build $(GO_ARGS) -o $@ ./cmd/$(@F)

cover.out: GO_ARGS := -coverprofile cover.out
cover.out: test

# Prevent intermediate files from being deleted
.SECONDARY:
