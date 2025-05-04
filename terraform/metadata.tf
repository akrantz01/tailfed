module "metadata" {
  source = "./modules/bucket"

  prefix = "tailfed-openid-metadata-"
}

resource "aws_iam_role" "metadata" {
  name = "TailfedMetadataApiGatewayAccess"

  assume_role_policy = data.aws_iam_policy_document.metadata_trust_policy.json
}

resource "aws_iam_role_policy" "metadata" {
  role = aws_iam_role.metadata.id

  name   = "ReadOnly"
  policy = data.aws_iam_policy_document.metadata.json
}

module "metadata_version" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.version.id

  object = "version.json"
  bucket = {
    name   = module.metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.metadata.arn
}

module "metadata_config" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.config.id

  object = "config.json"
  bucket = {
    name   = module.metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.metadata.arn
}

module "metadata_openid_discovery_document" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.openid_configuration.id

  object = "openid-configuration"
  bucket = {
    name   = module.metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.metadata.arn
}

module "metadata_jwks" {
  source = "./modules/apigateway-s3"

  rest_api_id = aws_api_gateway_rest_api.default.id
  resource_id = aws_api_gateway_resource.jwks.id

  object = "jwks.json"
  bucket = {
    name   = module.metadata.id
    region = var.region
  }

  role_arn = aws_iam_role.metadata.arn
}

data "aws_iam_policy_document" "metadata_trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "metadata" {
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [module.metadata.arn]
  }

  statement {
    effect  = "Allow"
    actions = ["s3:GetObject"]
    resources = [
      "${module.metadata.arn}/version.json",
      "${module.metadata.arn}/config.json",
      "${module.metadata.arn}/openid-configuration",
      "${module.metadata.arn}/jwks.json",
    ]
  }
}

