# These options can all be set using environment variables. The environment variable names follow the key names,
# prefixed with `TAILFED_` with any nesting replaced with `__` and dashes replaced with `-`. For example, the `log-level`
# key is read from the `TAILFED_LOG_LEVEL` environment variable.

# The minimum level to emit logs at
# Choices: panic, fatal, error, warn, info, debug, trace
# Default: info
log-level: info

# The path to write the generated web identity token to
# Default: /run/tailfed/token
path: /run/tailfed/token

# How often to refresh the token, specified using Go duration syntax
# Default: 1h
frequency: 1h

# The URL of the Tailfed API (required)
url: {{ .Url }}
