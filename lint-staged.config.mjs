export default {
    "*.go": [
        "golangci-lint run --fix",
        "go fmt",
    ]
}
