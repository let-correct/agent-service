###############################################################################
# variables.tf
###############################################################################

variable "aws_region" {
  description = "AWS region to deploy into"
  type        = string
  default     = "eu-west-2"
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "log_level" {
  description = "Lambda log level"
  type        = string
  default     = "INFO"
}

variable "google_client_id" {
  description = "Google OAuth client ID"
  type        = string
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  sensitive   = true
}

variable "google_redirect_url" {
  description = "Google OAuth redirect URL"
  type        = string
}

variable "google_workspace_client_id" {
  description = "Google OAuth client ID for Cognito federation (Google Workspace SSO)"
  type        = string
}

variable "google_workspace_client_secret" {
  description = "Google OAuth client secret for Cognito federation (Google Workspace SSO)"
  type        = string
  sensitive   = true
}

variable "cognito_callback_urls" {
  description = "Allowed redirect URIs after Google Workspace login"
  type        = list(string)
}

variable "arthur_client_id" {
  description = "Arthur OAuth client ID"
  type        = string
}

variable "arthur_client_secret" {
  description = "Arthur OAuth client secret"
  type        = string
  sensitive   = true
}

variable "arthur_redirect_url" {
  description = "Arthur OAuth redirect URL"
  type        = string
}

variable "arthur_auth_url" {
  description = "Arthur OAuth auth endpoint URL"
  type        = string
}

variable "arthur_token_url" {
  description = "Arthur OAuth token endpoint URL"
  type        = string
}
