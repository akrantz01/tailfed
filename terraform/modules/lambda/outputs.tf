output "id" {
  value       = aws_lambda_function.handler.id
  description = "The ID of the created Lambda function"
}

output "arn" {
  value       = aws_lambda_function.handler.arn
  description = "The ARN of the created Lambda function"
}

output "role_arn" {
  value       = aws_iam_role.handler.arn
  description = "The ARN of the Lambda execution role"
}

output "role_id" {
  value       = aws_iam_role.handler.id
  description = "The ID of the Lambda execution role"
}
