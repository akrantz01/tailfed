variable "arch" {
  type        = string
  description = "The runtime architecture to run on"
  default     = "amd64"

  validation {
    condition     = var.arch == "amd64" || var.arch == "arm64"
    error_message = "Architecture can either be 'amd64' or 'arm64'"
  }
}

variable "bucket" {
  type        = string
  description = "The S3 bucket containing the release artifacts"
}

variable "checksum" {
  type        = string
  description = "The source code checksum to deploy"
}

variable "environment" {
  type        = map(string)
  description = "The environment variables to set"
  default     = {}
}

variable "name" {
  type        = string
  description = "The name of the function"

  validation {
    condition     = contains(["initializer", "verifier", "finalizer", "generator"], var.name)
    error_message = "Unknown function binary (choices: initializer, verifier, finalizer, generator)"
  }
}

variable "timeout" {
  type        = number
  description = "The function timeout in seconds"
  default     = 5
}
