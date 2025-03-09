# Find all Go modules in the repository
GO_MODULES := $(shell find . -name "go.mod" -type f -exec dirname {} \; | sort)

.PHONY: all
all: fmt tidy lint test

.PHONY: fmt
fmt:
	@for module in $(GO_MODULES); do \
		echo "Formatting $$module"; \
		(cd $$module && go tool golines -m 120 --ignore-generated -w .); \
	done

.PHONY: test
test: generate
	@for module in $(GO_MODULES); do \
		echo "Testing $$module"; \
		(cd $$module && CGO_ENABLED=0 go test -race -cover ./...); \
	done

.PHONY: tidy
tidy:
	@for module in $(GO_MODULES); do \
		echo "Tidying $$module"; \
		(cd $$module && go mod tidy); \
	done

.PHONY: lint
lint:
	@for module in $(GO_MODULES); do \
		echo "Linting $$module"; \
		(cd $$module && go vet ./...); \
		(cd $$module && golangci-lint run); \
	done

.PHONY: generate
generate: clean
	@for module in $(GO_MODULES); do \
		echo "Generating mocks for $$module"; \
		(cd $$module && go tool mockery); \
	done

.PHONY: clean
clean:
	find . -name "mock_*.go" -type f -delete

.PHONY: install-tools
install-tools:
	@for module in $(GO_MODULES); do \
		echo "Installing tools for $$module"; \
		(cd $$module && go get -tool github.com/vektra/mockery/v2); \
		(cd $$module && go get -tool github.com/segmentio/golines@latest); \
	done
