default: test

test:
	@go test -v -cover ./...

.PHONY: test
