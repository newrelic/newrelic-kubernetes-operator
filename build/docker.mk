#
# Makefile fragment for Docker actions
#
DOCKER            ?= docker
DOCKER_FILE       ?= build/package/Docker
DOCKER_IMAGE      ?= $(PROJECT_NAME)

docker-login:
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: logging into docker hub"
	@if [ -z "${DOCKER_USERNAME}" ]; then \
		echo "Failure: DOCKER_USERNAME not set" ; \
		exit 1 ; \
	fi
	@if [ -z "${DOCKER_PASSWORD}" ]; then \
		echo "Failure: DOCKER_PASSWORD not set" ; \
		exit 1 ; \
	fi
	@echo "=== $(PROJECT_NAME) === [ docker-login     ]: username: '$$DOCKER_USERNAME'"
	@echo ${DOCKER_PASSWORD} | $(DOCKER) login -u ${DOCKER_USERNAME} --password-stdin


#
# These should be replaced by goreleaser
#

# Build the docker image
docker-build: test
	docker build -f $(DOCKER_FILE) -t ${DOCKER_IMG}

# Push the docker image
docker-push: docker-login
	$(DOCKER) push ${DOCKER_IMG}

.PHONY: docker-build docker-push docker-login
