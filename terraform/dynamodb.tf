resource "aws_dynamodb_table" "storage" {
  name         = "TailfedStorage"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "ID"

  attribute {
    name = "ID"
    type = "S"
  }

  ttl {
    enabled        = true
    attribute_name = "ExpiresAt"
  }

  # Use AWS-managed key
  server_side_encryption {
    enabled = false
  }
}
