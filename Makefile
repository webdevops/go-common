FIRST_GOPATH			:= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOLANGCI_LINT_BIN		:= $(FIRST_GOPATH)/bin/golangci-lint
GOSEC_BIN				:= $(FIRST_GOPATH)/bin/gosec

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
	go mod verify


#######################################
# quality checks
#######################################

.PHONY: check
check: vendor lint test

.PHONY: test
test:
	time go test ./...

.PHONY: lint
lint: $(GOLANGCI_LINT_BIN)
	time $(GOLANGCI_LINT_BIN) run --verbose --print-resources-usage

$(GOLANGCI_LINT_BIN):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(FIRST_GOPATH)/bin
