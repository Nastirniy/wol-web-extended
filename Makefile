# Default to local image name for development
# To use local: export IMAGE_NAME=wol-web-extended:local
# To publish: export IMAGE_NAME=nastirniy/wol-web-extended:latest

#IMAGE_NAME ?= wol-web-extended:local
IMAGE_NAME ?= nastirniy/wol-web-extended:latest
VERSION ?= latest

.PHONY: buildx buildx-podman build-local test push help

# Show available commands
help:
	@echo "WoL-Web Docker Build Commands"
	@echo "=============================="
	@echo ""
	@echo "Quick Start:"
	@echo "  1. Set your image name: export IMAGE_NAME=nastirniy/wol-web-extended:latest"
	@echo "  2. Build and push: make buildx"
	@echo ""
	@echo "Available targets:"
	@echo "  make buildx          - Build multi-arch (amd64,arm64) and push to Docker Hub"
	@echo "  make buildx-podman   - Build multi-arch with Podman and push"
	@echo "  make build-local     - Build for current platform only (fast)"
	@echo "  make push            - Push local build to Docker Hub"
	@echo "  make test            - Test container locally"
	@echo "  make help            - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  IMAGE_NAME - Docker image name (default: $(IMAGE_NAME))"
	@echo "  VERSION    - Image version tag (default: $(VERSION))"
	@echo ""
	@echo "Examples:"
	@echo "  make IMAGE_NAME=myuser/wol:v1.0.0 buildx"
	@echo "  make build-local && make push"

# Multi-platform build with Docker and push to registry
buildx:
	@echo "Building multi-architecture image: $(IMAGE_NAME)"
	docker buildx build --push \
		--platform linux/arm64,linux/amd64 \
		-t $(IMAGE_NAME) .
	@echo "Successfully built and pushed: $(IMAGE_NAME)"

# Multi-platform build with Podman
buildx-podman:
	@echo "Building multi-architecture image with Podman: $(IMAGE_NAME)"
	podman buildx build --jobs=2 \
		--platform=linux/arm64,linux/amd64 \
		--manifest=$(IMAGE_NAME) .
	podman manifest push $(IMAGE_NAME)
	@echo "Successfully built and pushed: $(IMAGE_NAME)"

# Local build for current architecture
build-local:
	@echo "Building local image: $(IMAGE_NAME)"
	docker build -t $(IMAGE_NAME) .
	@echo "Successfully built: $(IMAGE_NAME)"
	@echo "To push: make push"

# Push local build to registry
push:
	@echo "Pushing image: $(IMAGE_NAME)"
	docker push $(IMAGE_NAME)
	@echo "Successfully pushed: $(IMAGE_NAME)"

# Test container locally
test:
	@echo "Testing container: $(IMAGE_NAME)"
	@echo "Access at http://localhost:8090"
	@echo "Press Ctrl+C to stop"
	docker run --rm -p 8090:8090 --network=host --cap-add=NET_RAW --cap-add=NET_ADMIN $(IMAGE_NAME)
