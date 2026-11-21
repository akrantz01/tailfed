terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "8.11.0"
    }
  }
}

locals {
  zone = coalesce(var.zone, data.google_compute_zones.available.names[0])

  arm64 = can(regex("^(t2a|c4a|n4a)-", var.machine_type))
  image = local.arm64 ? "debian-cloud/debian-13-arm64" : "debian-cloud/debian-13"

  artifacts_url = "https://tailfed-artifacts.s3.us-east-1.amazonaws.com/${var.server_version}"

  tailscale_settings = merge(
    {
      hostname        = var.name
      state-directory = "/var/lib/tailfed-server"
      tags            = var.tailscale_tags
      authentication  = "federation"
      federation = {
        client-id = var.tailscale_client_id
        audience  = var.tailscale_audience
      }
    },
    var.tailscale_control_plane_url == null ? {} : { control-plane-url = var.tailscale_control_plane_url },
  )

  settings = merge(
    var.settings,
    {
      listener = {
        http   = true
        https  = var.tailscale_https
        funnel = var.tailscale_funnel
      }
      tailscale = local.tailscale_settings
    },
  )
}

data "google_compute_zones" "available" {
  project = var.project
  region  = var.region
  status  = "UP"

  depends_on = [google_project_service.services]
}

resource "google_project_service" "services" {
  for_each = var.enable_apis ? toset(["compute.googleapis.com", "iam.googleapis.com"]) : toset([])

  project            = var.project
  service            = each.value
  disable_on_destroy = false
}
