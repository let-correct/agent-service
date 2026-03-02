output "ecr_repository_url" {
  description = "ECR repository URL — you will need this to push images and later to configure the Lambda"
  value       = aws_ecr_repository.auth_lambda_ecr.repository_url
}

output "lambda_function_name" {
  description = "Lambda function name"
  value       = aws_lambda_function.auth_lambda.function_name
}

output "lambda_function_arn" {
  description = "Lambda function ARN"
  value       = aws_lambda_function.auth_lambda.arn
}

output "api_gateway_url" {
  description = "Base URL for the HTTP API — append your path to call the Lambda"
  value       = aws_apigatewayv2_stage.auth_lambda.invoke_url
}
