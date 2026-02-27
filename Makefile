push-image-to-ecr:
	./scripts/push-to-ecr.sh my-lambda $(git rev-parse --short HEAD)