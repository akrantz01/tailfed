output "id" {
  value       = aws_lambda_function.handler.id
  description = "The ID of the created Lambda function"
}

output "arn" {
  value       = aws_lambda_function.handler.arn
  description = "The ARN of the created Lambda function"
}

output "name" {
  value       = aws_lambda_function.handler.function_name
  description = "The name of the created Lambda function"
}

output "invoke_arn" {
  value       = aws_lambda_function.handler.invoke_arn
  description = "The ARN used by other AWS services to invoke the created Lambda function"
}

output "role_arn" {
  value       = aws_iam_role.handler.arn
  description = "The ARN of the Lambda execution role"
}

output "role_id" {
  value       = aws_iam_role.handler.id
  description = "The ID of the Lambda execution role"
}
