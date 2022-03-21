FIRST_GOPATH			:= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOLANGCI_LINT_BIN		:= $(FIRST_GOPATH)/bin/golangci-lint
GOSEC_BIN				:= $(FIRST_GOPATH)/bin/gosec

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: test
test:
	go test ./...

.PHONY: check-release
check-release: vendor lint gosec test

.PHONY: lint
lint: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run -E exportloopref,gofmt --timeout=30m

.PHONY: gosec
gosec: $(GOSEC_BIN)
	$(GOSEC_BIN) ./...

$(GOLANGCI_LINT_BIN):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

$(GOSEC_BIN):
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(FIRST_GOPATH)/bin
