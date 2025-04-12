variable "architecture" {
  type        = string
  description = "The Lambda architecture to run on"
  default     = "amd64"

  validation {
    condition     = var.architecture == "amd64" || var.architecture == "arm64"
    error_message = "Lambda only supports amd64- and arm64-based runtimes"
  }
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
