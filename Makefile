# Some things this makefile could make use of:
#
# - test coverage target(s)
# - profiler target(s)
#

BIN             = 9volt
OUTPUT_DIR      = build
TMP_DIR        := .tmp
RELEASE_VER    := $(shell git rev-parse --short HEAD)
DOCKER_IP       = $(shell docker info | grep -q moby && echo localhost || docker-machine ip)
NAME            = default
COVERMODE       = atomic

TEST_PACKAGES      := $(shell go list ./... | grep -v vendor | grep -v fakes | grep -v ftest)

.PHONY: help
.DEFAULT_GOAL := help

run: ## Run application (without building)
	go run *.go server -d -u -e http://localhost:2379

all: test build docker ## Test, build and docker image build

setup: installtools ## Install and setup tools

test: ## Perform tests
	go test -cover $(TEST_PACKAGES)

testv: ## Perform tests (with verbose flag)
	go test -v -cover $(TEST_PACKAGES)

test/race: ## Perform tests and enable the race detector
	go test -race -cover $(TEST_PACKAGES)

test/cover: ## Run all tests + open coverage report for all packages
	echo 'mode: $(COVERMODE)' > .coverage
	for PKG in $(TEST_PACKAGES); do \
		go test -coverprofile=.coverage.tmp -tags "integration" $$PKG; \
		grep -v -E '^mode:' .coverage.tmp >> .coverage; \
	done
	go tool cover -html=.coverage
	$(RM) .coverage .coverage.tmp

installtools: ## Install development related tools
	echo 'NOTE: NodeJS 6+ needs to be available to build 9volt'
	go get github.com/kardianos/govendor
	go get github.com/maxbrunsfeld/counterfeiter
	go get github.com/yvasiyarov/swagger
	go get github.com/rakyll/statik

installnode: ## Used by TravisCI
	rm -rf ~/.nvm && \
	git clone https://github.com/creationix/nvm.git ~/.nvm && \
	(cd ~/.nvm && git checkout `git describe --abbrev=0 --tags`) && \
	. ~/.nvm/nvm.sh && \
	nvm install 6

generate: ## Run generate for non-vendor packages only
	go list ./... | grep -v vendor | xargs go generate
	go fmt ./fakes/...

build: semvercheck clean build/linux build/darwin ## Build for linux and darwin (save to OUTPUT_DIR/BIN)

build/linux: semvercheck clean/linux build/ui ## Build for linux (save to OUTPUT_DIR/BIN)
	GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.version=$(RELEASE_VER) -X main.semver=$(SEMVER)" -o $(OUTPUT_DIR)/$(BIN)-linux .

build/darwin: semvercheck clean/darwin build/ui ## Build for darwin (save to OUTPUT_DIR/BIN)
	GOOS=darwin go build -a -installsuffix cgo -ldflags "-X main.version=$(RELEASE_VER) -X main.semver=$(SEMVER)" -o $(OUTPUT_DIR)/$(BIN)-darwin .

build/docker: semvercheck build/linux ## Build docker image
	docker build -t "9volt:$(RELEASE_VER)" .

build/docker-compose: semvercheck build/linux ## Build and start 9volt (and etcd) using docker-compose
	docker-compose up -d

build/docs: ## Build markdown docs from swagger comments
	swagger -apiPackage="github.com/9corp/9volt" -format=markdown -output=docs/api/README.md

build/ui: ui ## Build the UI (use nvm if available)
	(if [ -e ~/.nvm/nvm.sh ]; then . ~/.nvm/nvm.sh; fi; cd ui && npm install && npm run build)
	statik -src=./ui/dist

build/release: semvercheck build/linux build/darwin ## Prepare a build
	mkdir $(OUTPUT_DIR)/9volt-$(SEMVER)-darwin
	mkdir $(OUTPUT_DIR)/9volt-$(SEMVER)-linux
	mv $(OUTPUT_DIR)/$(BIN)-darwin $(OUTPUT_DIR)/9volt-$(SEMVER)-darwin/$(BIN)
	mv $(OUTPUT_DIR)/$(BIN)-linux $(OUTPUT_DIR)/9volt-$(SEMVER)-linux/$(BIN)
	cp -prf docs/example-configs $(OUTPUT_DIR)/9volt-$(SEMVER)-darwin/
	cp -prf docs/example-configs $(OUTPUT_DIR)/9volt-$(SEMVER)-linux/
	cd $(OUTPUT_DIR) && tar -czvf 9volt-$(SEMVER)-darwin.tgz 9volt-$(SEMVER)-darwin/
	cd $(OUTPUT_DIR) && tar -czvf 9volt-$(SEMVER)-linux.tgz 9volt-$(SEMVER)-linux/
	@echo "A new release has been created!"

build/release-travis: installnode installtools build/release ## Install node, tools, build

build/release-travis-docker: semvercheck dockercheck ## Used by Travis
	docker login -e $(DOCKER_EMAIL) -u $(DOCKER_USER) -p $(DOCKER_PASS)
	docker build -t "9corp/9volt:$(SEMVER)" -t "9corp/9volt:latest" . && \
	docker push 9corp/9volt:$(SEMVER)
	docker push 9corp/9volt:latest

build/release-docker: semvercheck build/linux ## Build, tag and push a docker image to dockerhubs
	docker build -t "9corp/9volt:$(SEMVER)" -t "9corp/9volt:latest" . && \
	docker push 9corp/9volt:$(SEMVER)
	docker push 9corp/9volt:latest

dockercheck:
ifeq ($(DOCKER_EMAIL),)
	$(error 'DOCKER_EMAIL' must be set)
else
ifeq ($(DOCKER_USER),)
	$(error 'DOCKER_USER' must be set)
else
ifeq ($(DOCKER_PASS),)
	$(error 'DOCKER_PASS' must be set)
endif
endif
endif

semvercheck:
ifeq ($(SEMVER),)
	$(error 'SEMVER' must be set)
endif

clean: clean/darwin clean/linux ## Remove all build artifacts

clean/darwin: ## Remove darwin build artifacts
	$(RM) $(OUTPUT_DIR)/$(BIN)-darwin

clean/linux: ## Remove linux build artifacts
	$(RM) $(OUTPUT_DIR)/$(BIN)-linux

ui/dev: ## Install NPM modules for ui and run development
	@echo "=============================================================="
	@echo "Make sure 9Volt is running in another window (go run *.go -d -u)."
	@echo "=============================================================="
	(cd ui && npm install && npm run dev)

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
