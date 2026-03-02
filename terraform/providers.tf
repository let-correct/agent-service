###############################################################################
# providers.tf
###############################################################################

terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  cloud {
    organization = "let-correct-viewing"

    workspaces {
      name = "let-correct-viewing"
    }
  }
}

provider "aws" {
  region = var.aws_region
}
