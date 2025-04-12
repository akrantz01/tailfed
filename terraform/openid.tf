module "openid_configuration" {
  source = "./modules/bucket"

  prefix = "tailfed-openid-configuration-"
}

resource "aws_iam_role" "openid_configuration" {
  name = "TailfedOpenIDConfigurationApiGateway"

  assume_role_policy = data.aws_iam_policy_document.openid_configuration_trust_policy.json
}

resource "aws_iam_policy" "openid_configuration" {
  name   = "TailfedOpenIDConfigurationReadOnly"
  policy = data.aws_iam_policy_document.openid_configuration.json
}

resource "aws_iam_role_policy_attachment" "openid_configuration" {
  role       = aws_iam_role.openid_configuration.id
  policy_arn = aws_iam_policy.openid_configuration.arn
}

module "openid_configuration_discovery_document" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.openid_configuration.id

  object = "openid-configuration"
  bucket = {
    name   = module.openid_configuration.id
    region = var.region
  }

  role_arn = aws_iam_role.openid_configuration.arn
}

module "openid_configuration_jwks" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.jwks.id

  object = "jwks.json"
  bucket = {
    name   = module.openid_configuration.id
    region = var.region
  }

  role_arn = aws_iam_role.openid_configuration.arn
}
