terraform {
  required_version = "~> 1.9.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

locals {
  # Translate from Go architecture names to Lambda architecture names
  lambda_arches = {
    amd64 = "x86_64"
    arm64 = "arm64"
  }
}

resource "aws_lambda_function" "handler" {
  function_name = "Tailfed${title(var.name)}"
  architectures = [local.lambda_arches[var.arch]]

  package_type     = "Zip"
  s3_bucket        = var.bucket
  s3_key           = "${var.name}.zip"
  source_code_hash = var.checksum

  runtime = "provided.al2023"
  handler = "bootstrap"

  role = aws_iam_role.handler.arn

  environment {
    variables = var.environment
  }

  logging_config {
    log_format = "Text"
    log_group  = aws_cloudwatch_log_group.handler.name
  }
}

resource "aws_cloudwatch_log_group" "handler" {
  name              = "/aws/lambda/tailfed/${var.name}"
  retention_in_days = 7
}
