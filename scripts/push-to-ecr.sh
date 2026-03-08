#!/usr/bin/env bash
# scripts/push-to-ecr.sh
# Usage: ./scripts/push-to-ecr.sh <image-tag> <cmd>
# Example: ./scripts/push-to-ecr.sh $(git rev-parse --short HEAD) auth
# Example: ./scripts/push-to-ecr.sh $(git rev-parse --short HEAD) calendar-sync-dispatcher

set -euo pipefail

IMAGE_TAG=${1:?Usage: push-to-ecr.sh <image-tag> <cmd>}
CMD=${2:?Usage: push-to-ecr.sh <image-tag> <cmd>}
REPO_NAME="${CMD//-/_}_ecr"
REGION="eu-west-2"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGISTRY="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
FULL_IMAGE="${REGISTRY}/${REPO_NAME}:${IMAGE_TAG}"
FULL_IMAGE_LATEST="${REGISTRY}/${REPO_NAME}:latest"

echo "Authenticating with ECR..."
aws ecr get-login-password --region "$REGION" \
  | docker login --username AWS --password-stdin "$REGISTRY"

echo "Building and pushing ${REPO_NAME}..."
docker buildx build \
  --platform linux/arm64 \
  --provenance=false \
  --build-arg CMD="$CMD" \
  --tag "$FULL_IMAGE" \
  --tag "$FULL_IMAGE_LATEST" \
  --push \
  .

echo "Done — image available at:"
echo "  ${FULL_IMAGE}"
