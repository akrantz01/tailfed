resource "aws_api_gateway_rest_api" "default" {
  name        = "tailfed"
  description = "The API for generating tailnet node ID tokens"

  endpoint_configuration {
    types = ["EDGE"]
  }
}

resource "aws_api_gateway_deployment" "default" {
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
      module.openid_configuration_discovery_document.requires_redeployment,
      module.openid_configuration_jwks.requires_redeployment,
      module.initializer_apigateway.requires_redeployment,
      module.finalizer_apigateway.requires_redeployment,
    ]))
  }
}

resource "aws_api_gateway_stage" "production" {
  rest_api_id   = aws_api_gateway_rest_api.default.id
  deployment_id = aws_api_gateway_deployment.default.id

  stage_name = "production"
  # TODO: enable throttling
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
