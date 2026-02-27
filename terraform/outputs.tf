###############################################################################
# outputs.tf
###############################################################################

output "ecr_repository_url" {
  description = "ECR repository URL — you will need this to push images and later to configure the Lambda"
  value       = aws_ecr_repository.auth_lambda_ecr.repository_url
}
