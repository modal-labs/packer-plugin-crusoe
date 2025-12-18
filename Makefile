NAME=crusoe
BINARY=packer-plugin-${NAME}

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: dev
dev: build
	mkdir -p ~/.packer.d/plugins/
	cp ${BINARY} ~/.packer.d/plugins/

.PHONY: test
test:
	go test ./... -v

.PHONY: install
install: build
	mkdir -p ~/.packer.d/plugins/
	mv ${BINARY} ~/.packer.d/plugins/

.PHONY: clean
clean:
	rm -f ${BINARY}

.PHONY: generate
generate:
	go generate ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: mod
mod:
	go mod download
	go mod tidy

