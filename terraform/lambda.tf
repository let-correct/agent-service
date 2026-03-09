###############################################################################
# Auth Lambda Function
###############################################################################

resource "aws_lambda_function" "auth_lambda" {
  function_name = "auth-lambda"
  role          = aws_iam_role.auth_lambda.arn

  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.auth_ecr.repository_url}:latest"
  architectures = ["arm64"]

  memory_size = 128
  timeout     = 30

  kms_key_arn = aws_kms_key.oauth_tokens.arn

  environment {
    variables = {
      LOG_LEVEL            = var.log_level
      TOKEN_TABLE_NAME     = aws_dynamodb_table.oauth_tokens.name
      STATE_TABLE_NAME     = aws_dynamodb_table.oauth_state.name
      KMS_KEY_ARN          = aws_kms_key.oauth_tokens.arn
      GOOGLE_CLIENT_ID     = var.google_client_id
      GOOGLE_CLIENT_SECRET = var.google_client_secret
      GOOGLE_REDIRECT_URL  = var.google_redirect_url
      ARTHUR_CLIENT_ID = var.arthur_client_id
      ARTHUR_CLIENT_SECRET = var.arthur_client_secret
      ARTHUR_REDIRECT_URL  = var.arthur_redirect_url
      ARTHUR_AUTH_URL = var.arthur_auth_url
      ARTHUR_TOKEN_URL  = var.arthur_token_url
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

###############################################################################
# Calendar Sync Worker Lambda Function
###############################################################################

resource "aws_lambda_function" "calendar_sync_worker" {
  function_name = "calendar-sync-worker"
  role          = aws_iam_role.calendar_sync_worker.arn

  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.calendar_sync_worker_ecr.repository_url}:latest"
  architectures = ["arm64"]

  memory_size = 256
  timeout     = 30

  kms_key_arn = aws_kms_key.oauth_tokens.arn

  environment {
    variables = {
      LOG_LEVEL        = var.log_level
      TOKEN_TABLE_NAME = aws_dynamodb_table.oauth_tokens.name
      KMS_KEY_ARN      = aws_kms_key.oauth_tokens.arn
      SYNC_TABLE_NAME  = aws_dynamodb_table.google_calendar_sync_tokens.name
      EVENT_BUS_NAME   = aws_cloudwatch_event_bus.let_correct.name
    }
  }

  logging_config {
    log_format = "JSON"
    log_group  = aws_cloudwatch_log_group.calendar_sync_worker.name
  }

  tags = var.tags

  depends_on = [
    aws_iam_role_policy_attachment.calendar_sync_worker_basic_execution,
    aws_cloudwatch_log_group.calendar_sync_worker,
  ]
}

resource "aws_lambda_event_source_mapping" "calendar_sync_worker_sqs" {
  event_source_arn                   = aws_sqs_queue.calendar_sync_worker.arn
  function_name                      = aws_lambda_function.calendar_sync_worker.arn
  batch_size                         = 10
  maximum_batching_window_in_seconds = 30
  function_response_types            = ["ReportBatchItemFailures"]
}

###############################################################################
# IAM — Calendar Sync Worker Execution Role
###############################################################################

data "aws_iam_policy_document" "calendar_sync_worker" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "calendar_sync_worker" {
  name               = "calendar-sync-worker-execution-role"
  assume_role_policy = data.aws_iam_policy_document.calendar_sync_worker.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "calendar_sync_worker_basic_execution" {
  role       = aws_iam_role.calendar_sync_worker.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "calendar_sync_worker_permissions" {
  name = "calendar-sync-worker-permissions"
  role = aws_iam_role.calendar_sync_worker.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem"]
        Resource = aws_dynamodb_table.oauth_tokens.arn
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = aws_dynamodb_table.google_calendar_sync_tokens.arn
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt"]
        Resource = aws_kms_key.oauth_tokens.arn
      },
      {
        Effect   = "Allow"
        Action   = ["events:PutEvents"]
        Resource = aws_cloudwatch_event_bus.let_correct.arn
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
        ]
        Resource = aws_sqs_queue.calendar_sync_worker.arn
      },
    ]
  })
}

resource "aws_cloudwatch_log_group" "calendar_sync_worker" {
  name              = "/aws/lambda/calendar_sync_worker"
  retention_in_days = 30
  tags              = var.tags
}

###############################################################################
# Cognito Pre-Token Generation Lambda
###############################################################################

resource "aws_lambda_function" "cognito_pre_token_gen" {
  function_name = "cognito-pre-token-gen"
  role          = aws_iam_role.cognito_pre_token_gen.arn

  package_type  = "Image"
  image_uri     = "${aws_ecr_repository.cognito_pre_token_gen_ecr.repository_url}:latest"
  architectures = ["arm64"]

  memory_size = 128
  timeout     = 5

  logging_config {
    log_format = "JSON"
    log_group  = aws_cloudwatch_log_group.cognito_pre_token_gen.name
  }

  tags = var.tags

  depends_on = [
    aws_iam_role_policy_attachment.cognito_pre_token_gen_basic_execution,
    aws_cloudwatch_log_group.cognito_pre_token_gen,
  ]
}

resource "aws_lambda_permission" "cognito_pre_token_gen" {
  statement_id  = "AllowCognitoInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cognito_pre_token_gen.function_name
  principal     = "cognito-idp.amazonaws.com"
  source_arn    = aws_cognito_user_pool.agents.arn
}

data "aws_iam_policy_document" "cognito_pre_token_gen" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "cognito_pre_token_gen" {
  name               = "cognito-pre-token-gen-execution-role"
  assume_role_policy = data.aws_iam_policy_document.cognito_pre_token_gen.json
  tags               = var.tags
}

resource "aws_iam_role_policy_attachment" "cognito_pre_token_gen_basic_execution" {
  role       = aws_iam_role.cognito_pre_token_gen.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_cloudwatch_log_group" "cognito_pre_token_gen" {
  name              = "/aws/lambda/cognito_pre_token_gen"
  retention_in_days = 30
  tags              = var.tags
}