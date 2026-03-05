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

  tags = var.tags
}

resource "aws_cognito_user_pool_domain" "agents" {
  domain       = "agent-service"
  user_pool_id = aws_cognito_user_pool.agents.id
}

resource "aws_cognito_identity_provider" "google" {
  user_pool_id  = aws_cognito_user_pool.agents.id
  provider_name = "Google"
  provider_type = "Google"

  provider_details = {
    client_id        = var.google_workspace_client_id
    client_secret    = var.google_workspace_client_secret
    authorize_scopes = "openid email profile"
  }

  attribute_mapping = {
    email    = "email"
    username = "sub"
  }
}

resource "aws_cognito_user_pool_client" "agents" {
  name         = "agent-service-client"
  user_pool_id = aws_cognito_user_pool.agents.id

  generate_secret = false

  supported_identity_providers = ["Google"]

  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["openid", "email", "profile"]

  callback_urls = var.cognito_callback_urls

  explicit_auth_flows = [
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  access_token_validity  = 60 # minutes (1 hour)
  refresh_token_validity = 30 # days

  token_validity_units {
    access_token  = "minutes"
    refresh_token = "days"
  }

  depends_on = [aws_cognito_identity_provider.google]
}
