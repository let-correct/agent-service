test:
	go test ./...

push-auth:
	./scripts/push-to-ecr.sh $$(git rev-parse --short HEAD) auth

push-all:
	push-auth