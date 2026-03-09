################
# Auth ECR
################

resource "aws_ecr_repository" "auth_ecr" {
  name                 = "auth_ecr"
  # image_tag_mutability = "IMMUTABLE" # Prevents overwriting existing tags

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = var.tags
}

resource "aws_ecr_lifecycle_policy" "auth" {
  repository = aws_ecr_repository.auth_ecr.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 1 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 1
      }
      action = { type = "expire" }
    }]
  })
}

############################
# Calendar Sync Dispatcher ECR
############################

resource "aws_ecr_repository" "calendar_sync_dispatcher_ecr" {
  name = "calendar_sync_dispatcher_ecr"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = var.tags
}

resource "aws_ecr_lifecycle_policy" "calendar_sync_dispatcher" {
  repository = aws_ecr_repository.calendar_sync_dispatcher_ecr.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 1 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 1
      }
      action = { type = "expire" }
    }]
  })
}

############################
# Calendar Sync Worker ECR
############################

resource "aws_ecr_repository" "calendar_sync_worker_ecr" {
  name = "calendar_sync_worker_ecr"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = var.tags
}

resource "aws_ecr_lifecycle_policy" "calendar_sync_worker" {
  repository = aws_ecr_repository.calendar_sync_worker_ecr.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 1 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 1
      }
      action = { type = "expire" }
    }]
  })
}

############################
# Cognito Pre-Token Gen ECR
############################

resource "aws_ecr_repository" "cognito_pre_token_gen_ecr" {
  name = "cognito_pre_token_gen_ecr"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = var.tags
}

resource "aws_ecr_lifecycle_policy" "cognito_pre_token_gen" {
  repository = aws_ecr_repository.cognito_pre_token_gen_ecr.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 1 images"
      selection = {
        tagStatus   = "any"
        countType   = "imageCountMoreThan"
        countNumber = 1
      }
      action = { type = "expire" }
    }]
  })
}

################
# Next Lambda ECR
################
