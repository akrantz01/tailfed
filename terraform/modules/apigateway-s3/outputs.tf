output "requires_redeployment" {
  value = sha1(jsonencode([
    aws_api_gateway_method.s3,
    aws_api_gateway_integration.s3,
    aws_api_gateway_method_response.s3,
    aws_api_gateway_integration_response.s3,
  ]))
  description = "A hash that changes whenever a re-deployment is required"
}
