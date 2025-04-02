export default {
    "*.go": [
        "./scripts/golangci-lint",
        "./scripts/gofmt",
    ],
    "*.tf": [
        "tofu fmt"
    ]
}
