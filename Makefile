# Makefile for NOFX Docker Image Management
# Usage:
#   make build          - Build both backend and frontend images
#   make tag            - Tag images for Docker Hub
#   make push           - Push images to Docker Hub
#   make release        - Build, tag, and push (complete workflow)
#   make clean          - Remove local images

# Variables
DOCKER_USERNAME = zhjiax
IMAGE_NAME = nofx
BACKEND_TAG = backend
FRONTEND_TAG = frontend
VERSION ?= latest

# Image names
LOCAL_BACKEND_IMAGE = nofx-nofx
LOCAL_FRONTEND_IMAGE = nofx-nofx-frontend
REMOTE_BACKEND_IMAGE = $(DOCKER_USERNAME)/$(IMAGE_NAME):$(BACKEND_TAG)
REMOTE_FRONTEND_IMAGE = $(DOCKER_USERNAME)/$(IMAGE_NAME):$(FRONTEND_TAG)

# Colors for output
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help build build-backend build-frontend tag tag-backend tag-frontend push push-backend push-frontend release clean login

# Default target
help:
	@echo "$(GREEN)NOFX Docker Image Management$(NC)"
	@echo ""
	@echo "Available targets:"
	@echo "  $(YELLOW)build$(NC)          - Build both backend and frontend images"
	@echo "  $(YELLOW)build-backend$(NC)  - Build backend image only"
	@echo "  $(YELLOW)build-frontend$(NC) - Build frontend image only"
	@echo "  $(YELLOW)tag$(NC)            - Tag images for Docker Hub"
	@echo "  $(YELLOW)push$(NC)           - Push images to Docker Hub"
	@echo "  $(YELLOW)release$(NC)        - Build, tag, and push (complete workflow)"
	@echo "  $(YELLOW)clean$(NC)          - Remove local Docker images"
	@echo "  $(YELLOW)login$(NC)          - Login to Docker Hub"
	@echo ""
	@echo "Variables:"
	@echo "  DOCKER_USERNAME = $(DOCKER_USERNAME)"
	@echo "  IMAGE_NAME      = $(IMAGE_NAME)"
	@echo "  VERSION         = $(VERSION)"

# Build targets
build: build-backend build-frontend
	@echo "$(GREEN)✓ All images built successfully$(NC)"

build-backend:
	@echo "$(YELLOW)Building backend image...$(NC)"
	docker compose build nofx
	@echo "$(GREEN)✓ Backend image built$(NC)"

build-frontend:
	@echo "$(YELLOW)Building frontend image...$(NC)"
	docker compose build nofx-frontend
	@echo "$(GREEN)✓ Frontend image built$(NC)"

# Tag targets
tag: tag-backend tag-frontend
	@echo "$(GREEN)✓ All images tagged successfully$(NC)"

tag-backend:
	@echo "$(YELLOW)Tagging backend image...$(NC)"
	docker tag $(LOCAL_BACKEND_IMAGE):latest $(REMOTE_BACKEND_IMAGE)
	@if [ "$(VERSION)" != "latest" ]; then \
		docker tag $(LOCAL_BACKEND_IMAGE):latest $(DOCKER_USERNAME)/$(IMAGE_NAME):$(BACKEND_TAG)-$(VERSION); \
		echo "$(GREEN)✓ Backend tagged as $(BACKEND_TAG) and $(BACKEND_TAG)-$(VERSION)$(NC)"; \
	else \
		echo "$(GREEN)✓ Backend tagged as $(BACKEND_TAG)$(NC)"; \
	fi

tag-frontend:
	@echo "$(YELLOW)Tagging frontend image...$(NC)"
	docker tag $(LOCAL_FRONTEND_IMAGE):latest $(REMOTE_FRONTEND_IMAGE)
	@if [ "$(VERSION)" != "latest" ]; then \
		docker tag $(LOCAL_FRONTEND_IMAGE):latest $(DOCKER_USERNAME)/$(IMAGE_NAME):$(FRONTEND_TAG)-$(VERSION); \
		echo "$(GREEN)✓ Frontend tagged as $(FRONTEND_TAG) and $(FRONTEND_TAG)-$(VERSION)$(NC)"; \
	else \
		echo "$(GREEN)✓ Frontend tagged as $(FRONTEND_TAG)$(NC)"; \
	fi

# Push targets
push: push-backend push-frontend
	@echo "$(GREEN)✓ All images pushed successfully$(NC)"

push-backend:
	@echo "$(YELLOW)Pushing backend image...$(NC)"
	docker push $(REMOTE_BACKEND_IMAGE)
	@if [ "$(VERSION)" != "latest" ]; then \
		docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(BACKEND_TAG)-$(VERSION); \
	fi
	@echo "$(GREEN)✓ Backend image pushed$(NC)"

push-frontend:
	@echo "$(YELLOW)Pushing frontend image...$(NC)"
	docker push $(REMOTE_FRONTEND_IMAGE)
	@if [ "$(VERSION)" != "latest" ]; then \
		docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(FRONTEND_TAG)-$(VERSION); \
	fi
	@echo "$(GREEN)✓ Frontend image pushed$(NC)"

# Complete release workflow
release: build tag push
	@echo ""
	@echo "$(GREEN)════════════════════════════════════════$(NC)"
	@echo "$(GREEN)✓ Release completed successfully!$(NC)"
	@echo "$(GREEN)════════════════════════════════════════$(NC)"
	@echo ""
	@echo "Images available at:"
	@echo "  - $(REMOTE_BACKEND_IMAGE)"
	@echo "  - $(REMOTE_FRONTEND_IMAGE)"
	@if [ "$(VERSION)" != "latest" ]; then \
		echo "  - $(DOCKER_USERNAME)/$(IMAGE_NAME):$(BACKEND_TAG)-$(VERSION)"; \
		echo "  - $(DOCKER_USERNAME)/$(IMAGE_NAME):$(FRONTEND_TAG)-$(VERSION)"; \
	fi
	@echo ""
	@echo "View on Docker Hub: https://hub.docker.com/r/$(DOCKER_USERNAME)/$(IMAGE_NAME)"

# Login to Docker Hub
login:
	@echo "$(YELLOW)Logging in to Docker Hub...$(NC)"
	docker login
	@echo "$(GREEN)✓ Logged in successfully$(NC)"

# Clean up local images
clean:
	@echo "$(YELLOW)Removing local NOFX images...$(NC)"
	-docker rmi $(LOCAL_BACKEND_IMAGE):latest
	-docker rmi $(LOCAL_FRONTEND_IMAGE):latest
	-docker rmi $(REMOTE_BACKEND_IMAGE)
	-docker rmi $(REMOTE_FRONTEND_IMAGE)
	@if [ "$(VERSION)" != "latest" ]; then \
		docker rmi $(DOCKER_USERNAME)/$(IMAGE_NAME):$(BACKEND_TAG)-$(VERSION) 2>/dev/null || true; \
		docker rmi $(DOCKER_USERNAME)/$(IMAGE_NAME):$(FRONTEND_TAG)-$(VERSION) 2>/dev/null || true; \
	fi
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

# List current images
list:
	@echo "$(YELLOW)Current NOFX images:$(NC)"
	@docker images | grep -E "nofx|REPOSITORY" || echo "No NOFX images found"

