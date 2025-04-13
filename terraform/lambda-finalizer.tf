module "finalizer" {
  source = "./modules/lambda"

  depends_on = [aws_s3_object_copy.artifacts["finalizer"]]

  name = "finalizer"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = local.artifact_hashes["finalizer"]

  environment = {
    TAILFED_LOG_LEVEL         = var.log_level
    TAILFED_SIGNING__AUDIENCE = var.audience
    TAILFED_SIGNING__KEY      = aws_kms_alias.signer.name
    TAILFED_SIGNING__VALIDITY = var.validity
    TAILFED_STORAGE__TABLE    = aws_dynamodb_table.storage.name
  }
}

resource "aws_iam_role_policy" "finalizer" {
  role = module.finalizer.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.finalizer.json
}

module "finalizer_apigateway" {
  source = "./modules/apigateway-lambda"

  rest_api = aws_api_gateway_rest_api.default
  resource = aws_api_gateway_resource.finalize

  function = module.finalizer
}
