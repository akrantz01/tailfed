output "id" {
  value       = aws_s3_bucket.bucket.id
  description = "The ID of the created bucket"
}

output "arn" {
  value       = aws_s3_bucket.bucket.arn
  description = "The ARN of the created bucket"
}
