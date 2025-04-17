locals {
  custom_domain = var.domain != null
  invoke_url    = local.custom_domain ? "https://${aws_api_gateway_domain_name.production[0].domain_name}" : aws_api_gateway_stage.production.invoke_url
}

resource "aws_api_gateway_rest_api" "default" {
  name        = "tailfed"
  description = "The API for generating tailnet node ID tokens"

  disable_execute_api_endpoint = local.custom_domain

  endpoint_configuration {
    types = ["EDGE"]
  }
}

resource "aws_api_gateway_deployment" "default" {
  depends_on = [
    module.openid_metadata_discovery_document,
    module.openid_metadata_jwks,
    module.initializer_apigateway,
    module.finalizer_apigateway,
  ]

  rest_api_id = aws_api_gateway_rest_api.default.id

  lifecycle {
    create_before_destroy = true
  }

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.well_known,
      aws_api_gateway_resource.openid_configuration,
      aws_api_gateway_resource.jwks,
      aws_api_gateway_resource.start,
      aws_api_gateway_resource.finalize,
      module.openid_metadata_discovery_document.requires_redeployment,
      module.openid_metadata_jwks.requires_redeployment,
      module.initializer_apigateway.requires_redeployment,
      module.finalizer_apigateway.requires_redeployment,
    ]))
  }
}

resource "aws_api_gateway_stage" "production" {
  rest_api_id   = aws_api_gateway_rest_api.default.id
  deployment_id = aws_api_gateway_deployment.default.id

  stage_name = "production"
}

resource "aws_api_gateway_method_settings" "global" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  stage_name  = aws_api_gateway_stage.production.stage_name
  method_path = "*/*"

  settings {
    throttling_rate_limit  = 50
    throttling_burst_limit = 10
  }
}

resource "aws_api_gateway_method_settings" "throttling" {
  for_each = toset(["/start/POST", "/finalize/POST"])

  rest_api_id = aws_api_gateway_rest_api.default.id
  stage_name  = aws_api_gateway_stage.production.stage_name
  method_path = each.key

  settings {
    throttling_rate_limit  = 10
    throttling_burst_limit = 3
  }
}

resource "aws_api_gateway_resource" "start" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  parent_id   = aws_api_gateway_rest_api.default.root_resource_id
  path_part   = "start"
}

resource "aws_api_gateway_resource" "finalize" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  parent_id   = aws_api_gateway_rest_api.default.root_resource_id
  path_part   = "finalize"
}

resource "aws_api_gateway_resource" "well_known" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  parent_id   = aws_api_gateway_rest_api.default.root_resource_id
  path_part   = ".well-known"
}

resource "aws_api_gateway_resource" "openid_configuration" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  parent_id   = aws_api_gateway_resource.well_known.id
  path_part   = "openid-configuration"
}

resource "aws_api_gateway_resource" "jwks" {
  rest_api_id = aws_api_gateway_rest_api.default.id
  parent_id   = aws_api_gateway_resource.well_known.id
  path_part   = "jwks.json"
}

resource "aws_api_gateway_domain_name" "production" {
  count = local.custom_domain ? 1 : 0

  domain_name     = var.domain.name
  certificate_arn = var.domain.certificate

  endpoint_configuration {
    types = ["EDGE"]
  }
}

resource "aws_api_gateway_base_path_mapping" "production" {
  count = local.custom_domain ? 1 : 0

  api_id      = aws_api_gateway_rest_api.default.id
  domain_name = aws_api_gateway_domain_name.production[count.index].domain_name
  stage_name  = aws_api_gateway_stage.production.stage_name
}
