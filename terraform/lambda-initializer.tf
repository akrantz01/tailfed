module "initializer" {
  source = "./modules/lambda"

  name = "initializer"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["initializer"].checksum_sha256

  environment = {
    TAILFED_LOG_LEVEL                      = var.log_level
    TAILFED_STORAGE__TABLE                 = aws_dynamodb_table.storage.name
    TAILFED_TAILSCALE__TAILNET             = var.tailscale.tailnet
    TAILFED_TAILSCALE__API_KEY             = var.tailscale.api_key
    TAILFED_TAILSCALE__OAUTH_CLIENT_ID     = var.tailscale.oauth.client_id
    TAILFED_TAILSCALE__OAUTH_CLIENT_SECRET = var.tailscale.oauth.client_secret
  }
}

resource "aws_iam_role_policy" "initializer" {
  role = module.initializer.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.initializer.json
}

module "initializer_apigateway" {
  source = "./modules/apigateway-lambda"

  rest_api = aws_api_gateway_rest_api.default
  resource = aws_api_gateway_resource.start

  function = module.initializer
}
