#!/usr/bin/env bash
# scripts/push-to-ecr.sh
# Usage: ./scripts/push-to-ecr.sh <repository-name> <image-tag>
# Example: ./scripts/push-to-ecr.sh my-lambda a3f5c12

set -euo pipefail

REPO_NAME=${1:?Usage: push-to-ecr.sh <repository-name> <image-tag>}
IMAGE_TAG=${2:?Usage: push-to-ecr.sh <repository-name> <image-tag>}
REGION=${AWS_REGION:-eu-west-2}

# Derive account ID and registry URL from AWS — no hardcoding needed
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