output "url" {
  value       = local.invoke_url
  description = "The base URL to interact with Tailfed"
  depends_on = [
    aws_lambda_invocation.generator,
    aws_api_gateway_stage.production,
    aws_api_gateway_base_path_mapping.production,
  ]
}

output "domain_endpoint" {
  value       = local.custom_domain ? aws_api_gateway_domain_name.production[0].cloudfront_domain_name : null
  description = "The CloudFront domain name to use with CNAME records (custom domains only)"
}
