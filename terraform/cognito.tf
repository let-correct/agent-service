###############################################################################
# Cognito User Pool — agent identities
###############################################################################
resource "aws_cognito_user_pool" "agents" {
  name = "agent-service-users"

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  password_policy {
    minimum_length                   = 8
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = true
    require_lowercase                = true
    temporary_password_validity_days = 7
  }

  tags = var.tags
}

resource "aws_cognito_user_pool_client" "agents" {
  name         = "agent-service-client"
  user_pool_id = aws_cognito_user_pool.agents.id

  generate_secret = false

  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  access_token_validity  = 60   # minutes (1 hour)
  refresh_token_validity = 30   # days

  token_validity_units {
    access_token  = "minutes"
    refresh_token = "days"
  }
}

output "cognito_user_pool_id" {
  value = aws_cognito_user_pool.agents.id
}

output "cognito_user_pool_client_id" {
  value = aws_cognito_user_pool_client.agents.id
}
