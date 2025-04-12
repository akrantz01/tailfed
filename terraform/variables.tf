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

variable "tailscale" {
  type = object({
    tailnet  = string
    auth_key = string

    api_key = optional(string)
    oauth = optional(object({
      client_id     = string
      client_secret = string
    }))
  })
  description = "The Tailscale tailent and API authentication method"

  validation {
    condition     = length(var.tailscale.tailnet) > 0
    error_message = "A tailnet is required"
  }

  validation {
    condition     = length(var.tailscale.auth_key) > 0
    error_message = "An auth key is required"
  }

  validation {
    condition = (
      (var.tailscale.api_key == null && var.tailscale.oauth != null) ||
      (var.tailscale.api_key != null && var.tailscale.oauth == null)
    )
    error_message = "Exactly one authentication method must be provided"
  }
}

variable "validity" {
  type        = string
  description = "How long a token should be valid for. Formatted as a Go duration string"
  default     = "1h"
}
