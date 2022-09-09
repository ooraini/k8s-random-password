IMG ?= ghcr.io/ooraini/k8s-random-password

.PHONY: docker-build
docker-build:
	docker build -t ${IMG}:$$(git describe --tags --abbrev=0) .

.PHONY: docker-push
docker-push:
	docker push ${IMG}:$$(git describe --tags --abbrev=0)
