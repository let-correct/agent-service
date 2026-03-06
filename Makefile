test:
	go test ./...

push-image-to-ecr:
	./scripts/push-to-ecr.sh $$(git rev-parse --short HEAD)