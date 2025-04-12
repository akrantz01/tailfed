output "requires_redeployment" {
  value = sha1(jsonencode([
    aws_api_gateway_method.method,
    aws_api_gateway_integration.integration,
  ]))
  description = "A hash that changes whenever a re-deployment is required"
}
