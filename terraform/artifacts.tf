locals {
  artifacts = toset(["initializer", "verifier", "finalizer", "generator"])
}

module "artifacts_proxy" {
  source = "./modules/bucket"

  prefix = "tailfed-artifacts-proxy-"
}

data "aws_s3_object" "artifacts" {
  provider = aws.release
  for_each = local.artifacts

  bucket = var.release_bucket
  key    = "${var.release_version}/${each.key}-${var.architecture}.zip"
}

resource "aws_s3_object_copy" "artifacts" {
  for_each = local.artifacts

  bucket = module.artifacts_proxy.id
  key    = "${each.key}.zip"
  source = "${var.release_bucket}/${var.release_version}/${each.key}-${var.architecture}.zip"

  checksum_algorithm = "SHA256"

  copy_if_match = data.aws_s3_object.artifacts[each.key].etag
}
