resource "google_service_account" "server" {
  project      = var.project
  account_id   = "${var.name}-server"
  display_name = "tailfed-server"
  description  = "Identity for the tailfed-server VM, exchanged for Tailscale auth keys via workload identity federation"

  depends_on = [google_project_service.services]
}

resource "google_compute_instance" "server" {
  project      = var.project
  name         = var.name
  zone         = local.zone
  machine_type = var.machine_type

  allow_stopping_for_update = true

  boot_disk {
    auto_delete = true

    initialize_params {
      image  = local.image
      size   = var.disk_size
      type   = var.disk_type
      labels = var.labels
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.tailfed.id
    stack_type = "IPV4_IPV6"

    access_config {
      network_tier = "STANDARD"
    }

    # External IPv6 is only available on the premium tier
    ipv6_access_config {
      network_tier = "PREMIUM"
    }
  }

  scheduling {
    preemptible         = false
    provisioning_model  = "STANDARD"
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
  }

  shielded_instance_config {
    enable_secure_boot          = true
    enable_vtpm                 = true
    enable_integrity_monitoring = true
  }

  service_account {
    email  = google_service_account.server.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    enable-oslogin         = "TRUE"
    block-project-ssh-keys = "TRUE"

    startup-script = file("${path.module}/files/startup.sh")

    tailfed-config        = jsonencode(local.settings)
    tailfed-artifacts-url = local.artifacts_url
  }

  labels = var.labels
}
