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
