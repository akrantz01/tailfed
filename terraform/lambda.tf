locals {
  artifacts = toset(["initializer", "verifier", "finalizer", "generator"])

  generator_input = jsonencode({
    issuer = aws_api_gateway_deployment.default.invoke_url
  })
}

# module "initializer" {
#   source = "./modules/lambda"
#
#   name = "initializer"
#
#   registry = local.registry
#   tag      = var.tag
# }

# module "verifier" {
#   source = "./modules/lambda"
#
#   name = "verifier"
#
#   registry = var.registry
#   tag      = var.tag
# }
#
# module "finalizer" {
#   source = "./modules/lambda"
#
#   name = "finalizer"
#
#   registry = var.registry
#   tag      = var.tag
# }

module "generator" {
  source = "./modules/lambda"

  name = "generator"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = aws_s3_object_copy.artifacts["generator"].checksum_sha256

  environment = {
    TAILFED_METADATA__BUCKET = module.openid_configuration.id
    TAILFED_SIGNING__KEY     = aws_kms_alias.signer.name
  }
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
