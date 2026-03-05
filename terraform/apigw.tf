###############################################################################
# API Gateway v2 (HTTP API)
###############################################################################
resource "aws_cloudwatch_log_group" "auth_lambda_api_gateway" {
  name              = "/aws/apigateway/auth-lambda"
  retention_in_days = 30
  tags              = var.tags
}

resource "aws_apigatewayv2_api" "auth_lambda" {
  name          = "auth-lambda-api"
  protocol_type = "HTTP"
  description   = "HTTP API for auth-lambda API"
  tags          = var.tags
}

resource "aws_apigatewayv2_stage" "auth_lambda" {
  api_id      = aws_apigatewayv2_api.auth_lambda.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.auth_lambda_api_gateway.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      routeKey       = "$context.routeKey"
      path           = "$context.path"
      status         = "$context.status"
      responseLength = "$context.responseLength"
      latency        = "$context.responseLatency"
      userAgent      = "$context.identity.userAgent"
      sourceIp       = "$context.identity.sourceIp"
      errorMessage   = "$context.error.message"
      integrationError = "$context.integrationErrorMessage"
    })
  }

  tags = var.tags
}

resource "aws_apigatewayv2_integration" "auth_lambda" {
  api_id                 = aws_apigatewayv2_api.auth_lambda.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.auth_lambda.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_authorizer" "cognito_jwt" {
  api_id           = aws_apigatewayv2_api.auth_lambda.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "cognito-jwt"

  jwt_configuration {
    issuer   = "https://cognito-idp.${var.aws_region}.amazonaws.com/${aws_cognito_user_pool.agents.id}"
    audience = [aws_cognito_user_pool_client.agents.id]
  }
}

resource "aws_apigatewayv2_route" "initiate_auth" {
  api_id             = aws_apigatewayv2_api.auth_lambda.id
  route_key          = "GET /auth/{provider}"
  target             = "integrations/${aws_apigatewayv2_integration.auth_lambda.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito_jwt.id
}

resource "aws_apigatewayv2_route" "exchange_code" {
  api_id             = aws_apigatewayv2_api.auth_lambda.id
  route_key          = "POST /auth/{provider}/callback"
  target             = "integrations/${aws_apigatewayv2_integration.auth_lambda.id}"
  authorization_type = "JWT"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito_jwt.id
}

# Grants API Gateway permission to invoke the Lambda function
resource "aws_lambda_permission" "auth_lambda_api_gateway" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.auth_lambda.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.auth_lambda.execution_arn}/*/*"
}
