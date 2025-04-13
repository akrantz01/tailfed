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
    TAILFED_SIGNING__KEY     = aws_kms_alias.signer.name
  }
}

resource "aws_iam_role_policy" "generator" {
  role = module.generator.role_id

  name   = "Lambda"
  policy = data.aws_iam_policy_document.generator.json
}

resource "aws_lambda_invocation" "generator" {
  function_name = module.generator.id

  input = local.generator_input

  lifecycle_scope = "CRUD"
  triggers = {
    updated = local.artifact_hashes["generator"]
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
