variable "rest_api_id" {
  type        = string
  description = "The ID of the REST API gateway to attach to"
}

variable "resource_id" {
  type        = string
  description = "The ID of the REST API resource to attach to"
}

variable "bucket" {
  type = object({
    name   = string
    region = string
  })
  description = "The name and region of the bucket to route to"
}

variable "object" {
  type        = string
  description = "The name of the object within the bucket to route to"
}

variable "role_arn" {
  type        = string
  description = "The ARN of the role to use for the S3 integration"
}
