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
