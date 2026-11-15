output "instance_name" {
  value       = module.tailfed.instance_name
  description = "The name of the VM, which is also its tailnet hostname"
}

output "reset_command" {
  value       = module.tailfed.reset_command
  description = "Resets the VM, which applies version and configuration changes"
}
