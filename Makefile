default: help

.PHONY: help
help:
	@echo "\nUsage: make [command]"
	@echo "Commands:"
	@echo "  - test: runs the unit tests and displays a code coverage report."
	@echo "  - example: runs the example app found in the _example directory.\n"


.PHONY: test
test:
	go clean -testcache
	go test . -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: example
example:
	go run _example/main.go