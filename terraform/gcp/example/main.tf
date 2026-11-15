terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "8.11.0"
    }
    tailscale = {
      source  = "tailscale/tailscale"
      version = "0.30.0"
    }
  }
}

provider "google" {
  project = var.project
}

provider "tailscale" {}

resource "tailscale_federated_identity" "tailfed" {
  description = "tailfed"
  scopes      = ["auth_keys"]
  tags        = var.tailscale_tags
  issuer      = "https://accounts.google.com"
  subject     = module.tailfed.service_account_unique_id
}

module "tailfed" {
  source = "../"

  project        = var.project
  server_version = var.server_version

  tailscale_client_id = tailscale_federated_identity.tailfed.id
  tailscale_audience  = tailscale_federated_identity.tailfed.audience
  tailscale_tags      = var.tailscale_tags

  settings    = var.settings
  ssh_via_iap = var.ssh_via_iap
}
