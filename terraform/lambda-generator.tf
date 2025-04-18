locals {
  generator_input = jsonencode({
    issuer = local.invoke_url
  })
}

module "generator" {
  source = "./modules/lambda"

  depends_on = [aws_s3_object_copy.artifacts["generator"]]

  name = "generator"
  arch = var.architecture

  bucket   = module.artifacts_proxy.id
  checksum = local.artifact_hashes["generator"]

  environment = {
    TAILFED_LOG_LEVEL        = var.log_level
    TAILFED_METADATA__BUCKET = module.openid_metadata.id
    TAILFED_SIGNING__KEY     = aws_kms_alias.signer.arn
  }

  policies = merge({ Lambda = data.aws_iam_policy_document.generator.json }, var.execution_role_policies)
}

resource "aws_lambda_invocation" "generator" {
  function_name = module.generator.id

  input = local.generator_input

  lifecycle_scope = "CRUD"
  triggers = {
    code_updated   = local.artifact_hashes["generator"]
    lambda_updated = module.generator.sha256
  }
}

data "aws_iam_policy_document" "generator" {
  statement {
    sid     = "Metadata"
    effect  = "Allow"
    actions = ["s3:PutObject"]
    resources = [
      "${module.openid_metadata.arn}/openid-configuration",
      "${module.openid_metadata.arn}/jwks.json",
    ]
  }

  statement {
    sid    = "Signer"
    effect = "Allow"
    actions = [
      "kms:DescribeKey",
      "kms:GetPublicKey",
    ]
    resources = [aws_kms_key.signer.arn]
  }
}
