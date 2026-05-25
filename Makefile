APP_NAME = restic-scheduler
VERSION ?= $(shell git describe --tags --dirty)
GOFILES = *.go
# Multi-arch targets are generated from this
TARGET_ALIAS = $(APP_NAME)-linux-amd64 $(APP_NAME)-linux-arm $(APP_NAME)-linux-arm64
TARGETS = $(addprefix dist/,$(TARGET_ALIAS))
CURRENT_GOARCH = $(shell go env GOARCH)

# Default make target will run tests
.DEFAULT_GOAL = test

# Build all static Minitor binaries
.PHONY: all
all: $(TARGETS)

# Build all static Linux Minitor binaries
.PHONY: all-linux
all-linux: $(filter dist/$(APP_NAME)-linux-%,$(TARGETS))

# Build restic-scheduler for the current machine
$(APP_NAME): $(GOFILES)
	@echo Version: $(VERSION)
	go build -ldflags '-X "main.version=$(VERSION)"' -o $(APP_NAME)

.PHONY: build
build: $(APP_NAME)

# Run all tests
.PHONY: test
test:
	go test -v -coverprofile=coverage.out # -short
	go tool cover -func=coverage.out

# Itest binary with coverage flag
test/$(APP_NAME)-linux-$(CURRENT_GOARCH):
	go build -o test/$(APP_NAME)-linux-$(CURRENT_GOARCH) -cover

# Build container with itest binary for coverage
.PHONY: itest-container
itest-container: test/$(APP_NAME)-linux-$(CURRENT_GOARCH)
	docker build --build-arg DIST=test --platform=linux/$(CURRENT_GOARCH) . -t $(APP_NAME)-itest

.PHONY: itest
itest: itest-container
	# Clean coverage dir
	mkdir -p ./coverage
	rm ./coverage/*
	# Run unit tests once so we can combine coverage
	go test -cover -test.gocoverdir=coverage
	# Run all itests
	./itest/run-once.sh
	./itest/run-schedule.sh
	./itest/run-config-reload.sh
	# Generate coverage text
	go tool covdata textfmt -i coverage/ -o coverage.out
	go tool cover -func=coverage.out

# Installs pre-commit hooks
.PHONY: install-hooks
install-hooks:
	pre-commit install --install-hooks

# Runs pre-commit checks on files
.PHONY: check
check:
	pre-commit run --all-files

.PHONY: clean
clean:
	rm -f ./$(APP_NAME)
	rm -fr ./coverage ./coverage.out
	rm -fr ./dist
	rm -fr ./data/* ./itest/data/* ./itest/repo/*
	rm -f ./test/restic-scheduler-*

## Multi-arch targets
$(TARGETS): $(GOFILES)
	@echo Version: $(VERSION)
	mkdir -p ./dist
	GOOS=$(word 3, $(subst -, ,$(@))) GOARCH=$(word 4, $(subst -, ,$(@))) CGO_ENABLED=0 \
		 go build -ldflags '-X "main.version=$(VERSION)"' -a -installsuffix nocgo \
		 -o $@

.PHONY: $(TARGET_ALIAS)
$(TARGET_ALIAS):
	$(MAKE) $(addprefix dist/,$@)
