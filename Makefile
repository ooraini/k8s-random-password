IMG ?= ghcr.io/ooraini/k8s-random-password

.PHONY: docker-build
docker-build:
	docker build -t ${IMG}:$$(git rev-parse HEAD) .

.PHONY: docker-push
docker-push:
	docker push ${IMG}:$$(git rev-parse HEAD)
