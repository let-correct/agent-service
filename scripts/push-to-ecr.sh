#!/usr/bin/env bash
# scripts/push-to-ecr.sh
# Usage: ./scripts/push-to-ecr.sh <image-tag>
# Example: ./scripts/push-to-ecr.sh $(git rev-parse --short HEAD)

set -euo pipefail

IMAGE_TAG=${1:?Usage: push-to-ecr.sh <image-tag>}
REPO_NAME="auth_lambda_ecr"
REGION="eu-west-2"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
FULL_IMAGE="${REGISTRY}/${REPO_NAME}:${IMAGE_TAG}"

echo "Authenticating with ECR..."
aws ecr get-login-password --region "$REGION" \
  | docker login --username AWS --password-stdin "$REGISTRY"

echo "Building ${FULL_IMAGE}..."
docker buildx build \
  --platform linux/arm64 \
  --tag "$FULL_IMAGE" \
  --push \
  .

echo "Done — image available at:"
echo "  ${FULL_IMAGE}"