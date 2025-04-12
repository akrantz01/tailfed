data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "generator" {
  statement {
    sid     = "Metadata"
    effect  = "Allow"
    actions = ["s3:PutObject"]
    resources = [
      "${module.openid_configuration.arn}/openid-configuration",
      "${module.openid_configuration.arn}/jwks.json",
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

data "aws_iam_policy_document" "generator_schedule_trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "generator_schedule" {
  statement {
    effect    = "Allow"
    actions   = ["lambda:InvokeFunction"]
    resources = [module.generator.arn]
  }
}

data "aws_iam_policy_document" "openid_configuration_trust_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["apigateway.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "openid_configuration" {
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [module.openid_configuration.arn]
  }

  statement {
    effect  = "Allow"
    actions = ["s3:GetObject"]
    resources = [
      "${module.openid_configuration.arn}/openid-configuration",
      "${module.openid_configuration.arn}/jwks.json",
    ]
  }
}

data "aws_iam_policy_document" "signer" {
  statement {
    sid       = "EnableIAMUserPermissions"
    effect    = "Allow"
    actions   = ["kms:*"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }
  }

  // TODO: add statements for finalizer and metadata generator
}
