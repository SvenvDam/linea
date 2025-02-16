.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test -v -cover ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run
