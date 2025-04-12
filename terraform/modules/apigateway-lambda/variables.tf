variable "function" {
  type = object({
    name       = string
    invoke_arn = string
  })
  description = "Details about the Lambda function to invoke"
}

variable "method" {
  type        = string
  description = "The HTTP method to route to"
  default     = "POST"
}

variable "rest_api" {
  type = object({
    id            = string
    execution_arn = string
  })
  description = "The REST API gateway to attach to"
}

variable "resource" {
  type = object({
    id   = string
    path = string
  })
  description = "The REST API resource to attach to"
}
