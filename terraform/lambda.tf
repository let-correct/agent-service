###############################################################################
# Auth Lambda Function
###############################################################################

resource "aws_lambda_function" "auth_lambda" {
  function_name = "auth-lambda"
  role          = aws_iam_role.auth_lambda.arn

  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.auth_lambda_ecr.repository_url}:latest"
  architectures = ["arm64"]

  memory_size = 128
  timeout     = 30

#   environment {
#     variables = merge(
#       { LOG_LEVEL = var.log_level },
#       var.environment_variables
#     )
#   }

  logging_config {
    log_format = "JSON"
    log_group  = aws_cloudwatch_log_group.auth_lambda.name
  }

  tags = var.tags

  depends_on = [
    aws_iam_role_policy_attachment.auth_lambda_basic_execution,
    aws_cloudwatch_log_group.auth_lambda,
  ]
}

###############################################################################
# IAM — Lambda Execution Role
###############################################################################
data "aws_iam_policy_document" "auth_lambda" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "auth_lambda" {
  name               = "auth-lambda-execution-role"
  assume_role_policy = data.aws_iam_policy_document.auth_lambda.json
  tags               = var.tags
}

# Grants Lambda permission to write logs to CloudWatch
resource "aws_iam_role_policy_attachment" "auth_lambda_basic_execution" {
  role       = aws_iam_role.auth_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Add any additional permissions your function needs here (S3, DynamoDB, SSM etc.)
# resource "aws_iam_role_policy" "lambda_custom" {
#   name = "auth-lambda-custom-policy"
#   role = aws_iam_role.auth_lambda.id

#   policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [
#       # {
#       #   Effect   = "Allow"
#       #   Action   = ["ssm:GetParameter"]
#       #   Resource = "arn:aws:ssm:${var.aws_region}:*:parameter/${var.function_name}/*"
#       # },
#     ]
#   })
# }

###############################################################################
# CloudWatch Log Group
# Created explicitly so we control retention. Without this, Lambda creates it
# automatically with no retention policy (logs kept forever).
###############################################################################
resource "aws_cloudwatch_log_group" "auth_lambda" {
  name              = "/aws/lambda/auth_lambda"
  retention_in_days = 30
  tags              = var.tags
}