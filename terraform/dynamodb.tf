###############################################################################
# OAuth Tokens DynamoDB Table
###############################################################################

resource "aws_dynamodb_table" "oauth_tokens" {
  name         = "oauth-tokens"
  billing_mode = "PAY_PER_REQUEST"

  hash_key  = "email"
  range_key = "provider"

  attribute {
    name = "email"
    type = "S"
  }

  attribute {
    name = "provider"
    type = "S"
  }

  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.oauth_tokens.arn
  }

  tags = var.tags
}

###############################################################################
# Google Calendar Sync Tokens DynamoDB Table
###############################################################################

resource "aws_dynamodb_table" "google_calendar_sync_tokens" {
  name         = "google-calendar-sync-tokens"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "email"

  attribute {
    name = "email"
    type = "S"
  }

  tags = var.tags
}

###############################################################################
# OAuth State DynamoDB Table
###############################################################################

resource "aws_dynamodb_table" "oauth_state" {
  name         = "oauth-state"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "state_value"

  attribute {
    name = "state_value"
    type = "S"
  }

  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  tags = var.tags
}
