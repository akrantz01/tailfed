variable "state_bucket" {
  type        = string
  description = "The bucket holding all the state"
}

variable "state_key" {
  type        = string
  description = "Where to store the Terraform state in the bucket"
}

variable "state_region" {
  type        = string
  description = "The region where the state resources live"
}
