all: build_macos build_linux
build_macos: bin/api-macos
build_linux: bin/api-linux

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: bin/api-macos
bin/api-macos:
	@echo "Building API for MacOS"
	@GOOS=darwin GOARCH=amd64 go build -o bin/api-amd64-darwin ./cmd/api

.PHONY: bin/api-linux
bin/api-linux:
	@echo "Building API for Linux"
	@GOOS=linux GOARCH=amd64 go build -o bin/api-amd64-linux ./cmd/api