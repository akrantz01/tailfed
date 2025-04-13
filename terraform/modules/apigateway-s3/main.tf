terraform {
  required_version = "~> 1.9.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

resource "aws_api_gateway_method" "s3" {
  rest_api_id = var.rest_api_id
  resource_id = var.resource_id

  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "s3" {
  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.s3.http_method

  type                    = "AWS"
  integration_http_method = "GET"
  uri                     = "arn:aws:apigateway:${var.bucket.region}:${var.bucket.name}.s3:path/${var.object}"
  credentials             = var.role_arn
  timeout_milliseconds    = 5000
}

resource "aws_api_gateway_integration_response" "s3" {
  depends_on = [aws_api_gateway_method_response.s3]

  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.s3.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET'"
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type'"

    "method.response.header.Content-Type"        = "integration.response.header.Content-Type"
    "method.response.header.Content-Length"      = "integration.response.header.Content-Length"
    "method.response.header.Content-Disposition" = "integration.response.header.Content-Disposition"
  }
}

resource "aws_api_gateway_method_response" "s3" {
  depends_on = [aws_api_gateway_integration.s3]

  rest_api_id = var.rest_api_id
  resource_id = var.resource_id
  http_method = aws_api_gateway_method.s3.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Headers" = true

    "method.response.header.Content-Type"        = true
    "method.response.header.Content-Length"      = true
    "method.response.header.Content-Disposition" = true
  }
}
