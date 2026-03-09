test:
	go test ./...

push-auth:
	./scripts/push-to-ecr.sh $$(git rev-parse --short HEAD) auth

push-calendar-sync-worker:
	./scripts/push-to-ecr.sh $$(git rev-parse --short HEAD) calendar-sync-worker

push-cognito-pre-token-gen:
	./scripts/push-to-ecr.sh $$(git rev-parse --short HEAD) cognito-pre-token-gen

push-all:
	$(MAKE) push-auth push-calendar-sync-worker push-cognito-pre-token-gen