src_files = $(shell go list -json -compiled -deps ./cmd/$(1)/ | jq --argjson module "$$(go list -m -json)" -rs 'map((.Dir | sub("^" + $$module.Dir; ".")) as $$relpath | select(.ImportPath | startswith($$module.Path)) | .CompiledGoFiles | map($$relpath + "/" + .)) | flatten | .[]')

.PHONY: build coverage help test

help:
	$(info help     - display this message)
	$(info build    - compile all the applications)
	$(info test     - run unit tests)
	$(info coverage - run unit tests with coverage)

build: out/client out/dev

test:
	go test $(TEST_ARGS) ./...

coverage: cover.out
	go tool cover -html cover.out

.SECONDEXPANSION:

cover.out: TEST_ARGS := -coverprofile cover.out
cover.out: test

out/%: $$(call src_files,$$(@F))
	go build -o $@ ./cmd/$(@F)
