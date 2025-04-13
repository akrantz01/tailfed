resource "aws_kms_key" "signer" {
  description              = "Token signing key for Tailfed"
  customer_master_key_spec = "ECC_NIST_P256"
  key_usage                = "SIGN_VERIFY"
  policy                   = data.aws_iam_policy_document.signer.json
}

resource "aws_kms_alias" "signer" {
  target_key_id = aws_kms_key.signer.id
  name          = "alias/tailfed"
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
}
