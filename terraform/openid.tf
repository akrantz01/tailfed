module "openid_metadata" {
  source = "./modules/bucket"

  prefix = "tailfed-openid-metadata-"
}

resource "aws_iam_role" "openid_metadata" {
  name = "TailfedOpenIDMetadataApiGatewayAccess"

  assume_role_policy = data.aws_iam_policy_document.openid_metadata_trust_policy.json
}

resource "aws_iam_role_policy" "openid_metadata" {
  role = aws_iam_role.openid_metadata.id

  name   = "ReadOnly"
  policy = data.aws_iam_policy_document.openid_metadata.json
}

module "openid_metadata_discovery_document" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.openid_configuration.id

  object = "openid-configuration"
  bucket = {
    name   = module.openid_metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.openid_metadata.arn
}

module "openid_metadata_jwks" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.jwks.id

  object = "jwks.json"
  bucket = {
    name   = module.openid_metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.openid_metadata.arn
}

data "aws_iam_policy_document" "openid_metadata_trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "openid_metadata" {
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [module.openid_metadata.arn]
  }

  statement {
    effect  = "Allow"
    actions = ["s3:GetObject"]
    resources = [
      "${module.openid_metadata.arn}/openid-configuration",
      "${module.openid_metadata.arn}/jwks.json",
    ]
  }
}

