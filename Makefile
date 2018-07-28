default: vet test

test:
	@go test -v -cover ./...

vet:
	@go vet ./...

.PHONY: test vet
