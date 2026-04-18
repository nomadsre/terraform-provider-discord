HOSTNAME   ?= registry.terraform.io
NAMESPACE  ?= nomadsre
NAME       ?= discord
BINARY     ?= terraform-provider-$(NAME)
VERSION    ?= 0.1.0
OS_ARCH    ?= $(shell go env GOOS)_$(shell go env GOARCH)

INSTALL_DIR := $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)

.PHONY: default
default: build

.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: install
install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY) $(INSTALL_DIR)/

.PHONY: test
test:
	go test -count=1 -timeout 120s ./...

.PHONY: testacc
testacc:
	TF_ACC=1 go test -count=1 -timeout 30m -v ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -s -w .
	terraform fmt -recursive ./examples/

.PHONY: docs
docs:
	go tool tfplugindocs generate -provider-name discord

.PHONY: tidy
tidy:
	go mod tidy
