###############################################################################
# main.tf
###############################################################################

resource "aws_ecr_repository" "auth_lambda_ecr" {
  name                 = "auth_lambda_ecr"
  # image_tag_mutability = "IMMUTABLE" # Prevents overwriting existing tags

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = var.tags
}

resource "aws_ecr_lifecycle_policy" "lambda" {
  repository = aws_ecr_repository.auth_lambda_ecr.name

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
