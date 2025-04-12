terraform {
  required_version = "~> 1.9.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

resource "aws_api_gateway_method" "method" {
  rest_api_id = var.rest_api.id
  resource_id = var.resource.id

  http_method   = var.method
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "integration" {
  rest_api_id = var.rest_api.id
  resource_id = var.resource.id
  http_method = aws_api_gateway_method.method.http_method

  type                    = "AWS_PROXY"
  integration_http_method = "POST"
  uri                     = var.function.invoke_arn
}

resource "aws_lambda_permission" "handler" {
  function_name = var.function.name

  statement_id = "AllowExecutionFromAPIGateway"
  action       = "lambda:InvokeFunction"
  principal    = "apigateway.amazonaws.com"

  source_arn = "${var.rest_api.execution_arn}/*/${aws_api_gateway_method.method.http_method}${var.resource.path}"
}
