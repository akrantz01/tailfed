variable "project" {
  type        = string
  description = "The Google Cloud project ID to deploy into"
}

variable "server_version" {
  type        = string
  description = "The tailfed-server version to deploy"
}

variable "tailscale_tags" {
  type        = list(string)
  default     = ["tag:tailfed"]
  description = "The tags to register the server with. They must exist in the tailnet policy's `tagOwners` and be granted the `funnel` node attribute."
}

variable "settings" {
  type = any
  default = {
    token = {
      profiles = {
        default = { audience = "sts.amazonaws.com" }
      }
    }
  }
  description = "Additional tailfed-server configuration. See the module's `settings` variable."
}

variable "ssh_via_iap" {
  type        = bool
  default     = false
  description = "Allow SSH to the VM through Identity-Aware Proxy"
}
