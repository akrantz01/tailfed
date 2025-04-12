locals {
  artifacts = toset(["initializer", "verifier", "finalizer", "generator"])

  generator_input = jsonencode({
    issuer = aws_api_gateway_deployment.default.invoke_url
  })
}

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

module "verifier" {
  source = "./modules/lambda"

  name = "verifier"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["verifier"].checksum_sha256

  environment = {
    TAILFED_LOG_LEVEL           = var.log_level
    TAILFED_STORAGE__TABLE      = aws_dynamodb_table.storage.name
    TAILFED_TAILSCALE__TAILNET  = var.tailscale.tailnet
    TAILFED_TAILSCALE__AUTH_KEY = var.tailscale.auth_key
  }
}

resource "aws_iam_role_policy" "verifier" {
  role = module.verifier.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.verifier.json
}

module "finalizer" {
  source = "./modules/lambda"

  name = "finalizer"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["finalizer"].checksum_sha256

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

module "generator" {
  source = "./modules/lambda"

  name = "generator"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["generator"].checksum_sha256

  environment = {
    TAILFED_LOG_LEVEL        = var.log_level
    TAILFED_METADATA__BUCKET = module.openid_configuration.id
    TAILFED_SIGNING__KEY     = aws_kms_alias.signer.name
  }
}

resource "aws_iam_role_policy" "generator" {
  role = module.generator.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.generator.json
}

resource "aws_lambda_invocation" "generator" {
  function_name   = module.generator.id
  lifecycle_scope = "CRUD"

  input = local.generator_input
}

module "artifacts_proxy" {
  source = "./modules/bucket"

  prefix = "tailfed-artifacts-proxy-"
}

resource "aws_s3_object_copy" "artifacts" {
  for_each = local.artifacts

  bucket = module.artifacts_proxy.id
  key    = "${each.key}.zip"
  source = "${var.release_bucket}/${var.release_version}/${each.key}-${var.architecture}.zip"

  checksum_algorithm = "SHA256"
}
