IMG ?= ghcr.io/ooraini/k8s-random-password

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG}:$$(git rev-parse HEAD) .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}:$$(git rev-parse HEAD)
