IMAGE ?= remotedialer-proxy
DOCKERFILE ?= Dockerfile.proxy
CONTEXT ?= .
TAG ?= latest
REPO ?= rancher

.PHONY: build push-image
build:
	@echo "Building image $(REPO)/$(IMAGE):$(TAG) for local use"
	docker buildx build \
		--tag $(REPO)/$(IMAGE):$(TAG) \
		--file $(DOCKERFILE) \
		--load \
		$(CONTEXT)

#push-image is used by the publish-image github action
push-image:
	@echo "Pushing image $(REPO)/$(IMAGE):$(TAG) for platforms $(TARGET_PLATFORMS)"
	docker buildx build \
		--tag $(REPO)/$(IMAGE):$(TAG) \
		--file $(DOCKERFILE) \
		--platform $(TARGET_PLATFORMS) \
		--push \
		$(IID_FILE_FLAG) \
		$(CONTEXT)
