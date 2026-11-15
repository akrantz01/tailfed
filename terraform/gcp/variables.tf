variable "project" {
  type        = string
  description = "The Google Cloud project ID to deploy into"
}

variable "server_version" {
  type        = string
  description = "The tailfed-server version to download from the tailfed-artifacts bucket (e.g. `2.0.0` or `nightly`)"
}

variable "tailscale_client_id" {
  type        = string
  description = <<-EOT
    The client ID of the Tailscale federated identity the server authenticates with. Create it with
    the Google Cloud issuer and the `service_account_unique_id` output as the subject.
    https://tailscale.com/docs/features/workload-identity-federation#configure-federated-identities-in-the-admin-console
  EOT
}

variable "tailscale_audience" {
  type        = string
  description = <<-EOT
    The audience of the Tailscale federated identity, in the form `api.tailscale.com/<client ID>`.
    https://tailscale.com/docs/features/workload-identity-federation#generated-audience-format
  EOT
}

variable "tailscale_tags" {
  type        = list(string)
  description = <<-EOT
    The tags to register the node with. They must be granted to the federated identity.
    https://tailscale.com/docs/features/tags
  EOT

  validation {
    condition     = length(var.tailscale_tags) > 0
    error_message = "At least one tag is required."
  }
}

variable "tailscale_funnel" {
  type        = bool
  default     = true
  description = <<-EOT
    Expose the token verification routes publicly on port 443 using Tailscale Funnel, which is
    required for verifiers outside the tailnet, such as AWS. The tailnet must have MagicDNS and HTTPS
    certificates enabled, and the policy must grant the `funnel` node attribute to the server's tags.
    https://tailscale.com/docs/features/tailscale-funnel#requirements-and-limitations
  EOT
}

variable "tailscale_control_plane_url" {
  type        = string
  default     = null
  description = "The Tailscale-compatible control plane to connect to. Defaults to Tailscale's."
}

variable "settings" {
  type        = any
  default     = {}
  description = <<-EOT
    Additional tailfed-server configuration. See `tailfed-server.example.yml`.
    The `listener` and `tailscale` keys are managed by the module and cannot be set here.
  EOT

  validation {
    condition     = !can(var.settings.listener) && !can(var.settings.tailscale)
    error_message = "The listener and tailscale settings are managed by the module; use the tailscale_* variables instead."
  }
}

variable "name" {
  type        = string
  default     = "tailfed"
  description = "Name used for the VM (and its tailnet hostname) and as a prefix for other resources"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{0,20}[a-z0-9]$", var.name))
    error_message = "The name must be 2-22 characters of lowercase letters, digits, and hyphens, starting with a letter."
  }
}

variable "region" {
  type        = string
  default     = "us-central1"
  description = "The region to deploy into. Only us-west1, us-central1, and us-east1 are covered by the Always Free tier."
}

variable "zone" {
  type        = string
  default     = null
  description = "The zone to deploy the VM into. Defaults to the first available zone in the region."

  validation {
    condition     = var.zone == null || startswith(coalesce(var.zone, "-"), "${var.region}-")
    error_message = "The zone must be within the selected region."
  }
}

variable "machine_type" {
  type        = string
  default     = "e2-micro"
  description = "The VM machine type. Only e2-micro is covered by the Always Free tier."
}

variable "disk_type" {
  type        = string
  default     = "pd-standard"
  description = "The boot disk type. Only pd-standard is covered by the Always Free tier."
}

variable "disk_size" {
  type        = number
  default     = 10
  description = "The boot disk size in GB. The Always Free tier covers up to 30 GB of standard persistent disk."

  validation {
    condition     = var.disk_size >= 10
    error_message = "The Debian 13 image requires at least 10 GB."
  }
}

variable "ssh_via_iap" {
  type        = bool
  default     = false
  description = "Allow SSH to the VM through Identity-Aware Proxy TCP forwarding (`gcloud compute ssh --tunnel-through-iap`)"
}

variable "enable_apis" {
  type        = bool
  default     = true
  description = "Enable the required Google Cloud APIs in the project"
}

variable "labels" {
  type        = map(string)
  default     = {}
  description = "Labels to apply to all supported resources"
}
