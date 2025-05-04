variable "architecture" {
  type        = string
  description = "The Lambda architecture to run on"
  default     = "amd64"

  validation {
    condition     = var.architecture == "amd64" || var.architecture == "arm64"
    error_message = "Lambda only supports amd64- and arm64-based runtimes"
  }
}

variable "audience" {
  type        = string
  description = "The token audience to issue for"
  default     = "sts.amazonaws.com"
}

variable "domain" {
  type = object({
    name        = string
    certificate = string
  })
  description = "The custom domain and associated ACM certificate to use"
  default     = null
}

variable "execution_role_policies" {
  type        = map(string)
  description = "Additional policies to attach to the Lambda execution roles"
  default     = {}
}

variable "log_level" {
  type        = string
  description = "The level for functions to log at"
  default     = "info"

  validation {
    condition     = contains(["panic", "fatal", "error", "warn", "info", "debug", "trace"], var.log_level)
    error_message = "Unknown log level (options: panic, fatal, error, warn, info, debug, trace)"
  }
}

variable "region" {
  type        = string
  description = "The region to deploy resource to"
}

variable "release_bucket" {
  type        = string
  description = "The bucket to pull release artifacts from"
  default     = "tailfed-artifacts"
}

variable "release_version" {
  type        = string
  description = "The release version to deploy, must exist within the bucket"
}

variable "tailscale_backend" {
  type        = string
  description = "The Tailscale backend implementation to use"
  default     = "tailscale"

  validation {
    condition     = contains(["tailscale", "headscale"], var.tailscale_backend)
    error_message = "Unknown tailscale backend (options: tailscale, headscale)"
  }
}

variable "tailscale_base_url" {
  type        = string
  description = "The base URL of the Tailscale backend"
  default     = "https://api.tailscale.com"

  validation {
    condition     = can(regex("^https?://[\\w.-]+\\.[a-zA-Z]{2,}(:[0-9]{1,5})?(/.*)?$", var.tailscale_base_url))
    error_message = "Must be a valid HTTP or HTTPS URL with a properly formatted domain"
  }
}

variable "tailscale_tls_mode" {
  type        = string
  description = "The TLS mode to use when connecting to the Tailscale control plane API"
  default     = "full"

  validation {
    condition     = contains(["none", "insecure", "full"], var.tailscale_tls_mode)
    error_message = "Unknown TLS mode (options: none, insecure, full)"
  }
}

variable "tailscale_tailnet" {
  type        = string
  description = "The name of the tailnet to validate against"

  validation {
    condition     = length(var.tailscale_tailnet) > 0
    error_message = "A tailnet is required"
  }
}

variable "tailscale_auth_key" {
  type        = string
  description = "The Tailscale auth key used to connect the verifier to your tailnet"

  validation {
    condition     = length(var.tailscale_auth_key) > 0
    error_message = "An auth key is required"
  }
}

# terraform-docs-ignore
variable "__tailscale_api_authentication" {
  type        = string
  description = "Dummy variable for validating the Tailscale API authentication method"
  default     = "(dummy)"

  validation {
    condition     = var.__tailscale_api_authentication == "(dummy)"
    error_message = "This variable should not be changed"
  }

  validation {
    condition = var.__tailscale_api_authentication == "(dummy)" && (
      (var.tailscale_api_key != null && var.tailscale_oauth == null) ||
      (var.tailscale_oauth != null && var.tailscale_api_key == null)
    )
    error_message = "Exactly one API authentication method can be enabled"
  }
}

variable "tailscale_api_key" {
  type        = string
  description = "The API key used to authenticate with the Tailscale API"
  nullable    = true
  default     = null

  validation {
    condition     = var.tailscale_api_key == null || var.tailscale_api_key != ""
    error_message = "Tailscale API key is required when non-null"
  }
}

variable "tailscale_oauth" {
  type = object({
    client_id     = string
    client_secret = string
  })
  description = "The OAuth client credentials used to authenticate with the Tailscale API"
  nullable    = true
  default     = null

  validation {
    condition     = var.tailscale_oauth == null || var.tailscale_oauth.client_id != ""
    error_message = "Tailscale OAuth client ID is required when non-null"
  }

  validation {
    condition     = var.tailscale_oauth == null || var.tailscale_oauth.client_secret != ""
    error_message = "Tailscale OAuth client secret is required when non-null"
  }

  validation {
    condition     = !(var.tailscale_backend == "headscale" && var.tailscale_oauth != null)
    error_message = "Headscale does not support OAuth-based authentication"
  }
}

variable "validity" {
  type        = string
  description = "How long a token should be valid for. Formatted as a Go duration string"
  default     = "1h"
}
