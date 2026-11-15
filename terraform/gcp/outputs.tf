output "instance_name" {
  value       = google_compute_instance.server.name
  description = "The name of the VM, which is also its tailnet hostname"
}

output "zone" {
  value       = google_compute_instance.server.zone
  description = "The zone the VM is deployed in"
}

output "service_account_email" {
  value       = google_service_account.server.email
  description = "The email of the VM's service account"
}

output "service_account_unique_id" {
  value       = google_service_account.server.unique_id
  description = "The subject of the VM's identity tokens, used when configuring a Tailscale federated identity (issuer: https://accounts.google.com)"
}

output "reset_command" {
  value       = "gcloud compute instances reset ${google_compute_instance.server.name} --project ${var.project} --zone ${google_compute_instance.server.zone}"
  description = "Resets the VM, which applies version and configuration changes"
}
