###############################################################################
# KMS Key — OAuth Token Encryption
###############################################################################

resource "aws_kms_key" "oauth_tokens" {
  description             = "Encrypts OAuth access and refresh tokens stored in DynamoDB"
  deletion_window_in_days = 7
  enable_key_rotation     = true
  tags                    = var.tags
}

resource "aws_kms_alias" "oauth_tokens" {
  name          = "alias/auth-lambda-oauth-tokens"
  target_key_id = aws_kms_key.oauth_tokens.key_id
}
