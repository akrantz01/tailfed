resource "aws_s3_bucket" "openid_configuration" {
  bucket_prefix = "tailfed-openid-configuration-"
  force_destroy = true
}

resource "aws_s3_bucket_ownership_controls" "openid_configuration" {
  bucket = aws_s3_bucket.openid_configuration.id
  rule {
    object_ownership = "BucketOwnerPreferred"
  }
}

resource "aws_s3_bucket_acl" "openid_configuration" {
  depends_on = [aws_s3_bucket_ownership_controls.openid_configuration]

  bucket = aws_s3_bucket.openid_configuration.id
  acl    = "private"
}

resource "aws_s3_bucket_public_access_block" "openid_configuration" {
  bucket = aws_s3_bucket.openid_configuration.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "openid_configuration" {
  bucket = aws_s3_bucket.openid_configuration.id

  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_iam_role" "openid_configuration" {
  name = "TailfedOpenIDConfigurationApiGateway"

  assume_role_policy = data.aws_iam_policy_document.openid_configuration_trust_policy.json
}

resource "aws_iam_policy" "openid_configuration" {
  name   = "TailfedOpenIDConfigurationReadOnly"
  policy = data.aws_iam_policy_document.openid_configuration.json
}

resource "aws_iam_role_policy_attachment" "openid_configuration" {
  role       = aws_iam_role.openid_configuration.id
  policy_arn = aws_iam_policy.openid_configuration.arn
}
