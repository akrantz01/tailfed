resource "google_compute_network" "tailfed" {
  project                 = var.project
  name                    = var.name
  auto_create_subnetworks = false

  depends_on = [google_project_service.services]
}

resource "google_compute_subnetwork" "tailfed" {
  project = var.project
  name    = var.name
  region  = var.region
  network = google_compute_network.tailfed.id

  stack_type       = "IPV4_IPV6"
  ip_cidr_range    = "10.0.0.0/28"
  ipv6_access_type = "EXTERNAL"
}

locals {
  # A firewall rule cannot mix address families
  # https://docs.cloud.google.com/iap/docs/using-tcp-forwarding#create-firewall-rule
  iap_source_ranges = {
    ipv4 = "35.235.240.0/20"
    ipv6 = "2600:2d00:1:7::/64"
  }
}

resource "google_compute_firewall" "iap_ssh" {
  for_each = var.ssh_via_iap ? local.iap_source_ranges : {}

  project = var.project
  name    = "${var.name}-allow-iap-ssh-${each.key}"
  network = google_compute_network.tailfed.id

  direction               = "INGRESS"
  source_ranges           = [each.value]
  target_service_accounts = [google_service_account.server.email]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}
