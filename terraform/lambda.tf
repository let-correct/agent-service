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

  kms_key_arn = aws_kms_key.oauth_tokens.arn

  environment {
    variables = {
      LOG_LEVEL            = var.log_level
      STATE_TABLE_NAME     = aws_dynamodb_table.oauth_state.name
      GOOGLE_CLIENT_ID     = var.google_client_id
      GOOGLE_CLIENT_SECRET = var.google_client_secret
      GOOGLE_REDIRECT_URL  = var.google_redirect_url
    }
  }

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

resource "aws_iam_role_policy" "auth_lambda_permissions" {
  name = "auth-lambda-permissions"
  role = aws_iam_role.auth_lambda.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:GetItem"]
        Resource = aws_dynamodb_table.oauth_tokens.arn
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:DeleteItem"]
        Resource = aws_dynamodb_table.oauth_state.arn
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:Encrypt", "kms:GenerateDataKey"]
        Resource = aws_kms_key.oauth_tokens.arn
      }
    ]
  })
}

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