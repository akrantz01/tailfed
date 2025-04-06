GO_ARGS :=

LAMBDA_BINARIES := finalizer initializer verifier
STANDARD_BINARIES := client dev

# Directory for build artifacts
OUT_DIR := out
ZIP_DIR := $(OUT_DIR)/lambda

.PHONY: all build build-client build-dev build-lambda clean coverage help test zip-lambda

help:
	$(info help         - display this message)
	$(info build        - compile all the applications)
	$(info build-client - build only the client application)
	$(info build-dev    - build only the dev application)
	$(info build-lambda - build only the lambda applications)
	$(info zip-lambda   - package lambda functions as zip archives)
	$(info test         - run unit tests)
	$(info coverage     - run unit tests with coverage)
	$(info clean        - remove build artifacts)
	$(info all          - build everything for deployment)

all: build zip-lambda

build: build-client build-dev build-lambda

build-client: $(addprefix $(OUT_DIR)/,$(STANDARD_BINARIES))

build-dev: $(OUT_DIR)/dev

build-lambda: GO_ARGS := -tags lambda.norpc
build-lambda: export CGO_ENABLED := 0
build-lambda: $(addprefix $(OUT_DIR)/,$(LAMBDA_BINARIES))

zip-lambda: build-lambda $(addprefix $(ZIP_DIR)/,$(addsuffix .zip,$(LAMBDA_BINARIES)))

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

# Lambda ZIP packaging rule
$(ZIP_DIR)/%.zip: $(OUT_DIR)/% | $(ZIP_DIR)
	cp $(OUT_DIR)/$(<F) $(OUT_DIR)/bootstrap
	cd $(OUT_DIR) && zip -FS -j ../$(ZIP_DIR)/$(@F) bootstrap
	rm $(OUT_DIR)/bootstrap

cover.out: GO_ARGS := -coverprofile cover.out
cover.out: test

# Prevent intermediate files from being deleted
.SECONDARY: