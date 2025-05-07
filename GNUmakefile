export GOPATH_BIN := $(shell go env GOPATH)/bin
export PATH := $(GOPATH_BIN):$(PATH)

TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=vsphere

default: build

build: fmtcheck
	go install

test: fmtcheck
	go test $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 360m

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

tools:
	go install -mod=mod github.com/katbyte/terrafmt

docs-check:
	@echo "==> Checking structure..."
	@sh -c "'$(CURDIR)/scripts/docscheck.sh'"

docs-hcl-lint: tools
	@echo "==> Checking HCL formatting..."
	@$(GOPATH_BIN)/terrafmt diff ./docs --check --pattern '*.md' --quiet || (echo; echo "Unexpected HCL differences. Run 'make docs-hcl-fix'."; exit 1)

docs-hcl-fix: tools
	@echo "==> Applying HCL formatting..."
	@$(GOPATH_BIN)/terrafmt fmt ./docs --pattern '*.md'

.PHONY: build test testacc fmt fmtcheck test-compile tools docs-check docs-hcl-lint docs-hcl-fix

