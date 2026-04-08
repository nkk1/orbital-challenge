.PHONY: all
all: help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.8.0

OAPI_CODEGEN := $(LOCALBIN)/oapi-codegen

KIND ?= kind
KIND_BIN = $(LOCALBIN)/kind

##@ Tools

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	@test -f .custom-gcl.yml && { \
		echo "Building custom golangci-lint with plugins..." && \
		$(GOLANGCI_LINT) custom --destination $(LOCALBIN) --name golangci-lint-custom && \
		mv -f $(LOCALBIN)/golangci-lint-custom $(GOLANGCI_LINT); \
	} || true

.PHONY: oapi
oapi: $(OAPI_CODEGEN) ## Download oapi-codegen locally if necessary.
$(OAPI_CODEGEN): $(LOCALBIN)
	$(call go-install-tool,$(OAPI_CODEGEN),github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen,latest)

.PHONY: kind
kind: $(KIND_BIN) ## Download kind locally if necessary.
$(KIND_BIN): $(LOCALBIN)
	curl --silent -Lo kind https://kind.sigs.k8s.io/dl/latest/kind-linux-$$(go env GOARCH)
	chmod +x kind
	mv kind $(KIND_BIN)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	"$(GOLANGCI_LINT)" run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	"$(GOLANGCI_LINT)" config verify

.PHONY: test
test: fmt vet ## Run tests.
	go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: run
run: build ## Build first 
	bin/orbital-service

.PHONY: smoke
smoke:
	go test -tags smoke -v ./server/api/...
	
##@ Build

.PHONY: oapi-gen
oapi-gen: oapi
	$(OAPI_CODEGEN) --config server/api/openapi-codegen.yaml ./server/api/openapi.yaml
	go mod tidy

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/orbital-service main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: docker-build kind ## Push docker image to kind	
	$(KIND) load docker-image ${IMG}


# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f "$(1)" ;\
GOBIN="$(LOCALBIN)" go install $${package} ;\
mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)" ;\
} ;\
ln -sf "$$(realpath "$(1)-$(3)")" "$(1)"
endef


