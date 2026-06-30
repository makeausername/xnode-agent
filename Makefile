BINARY := xnode
BIN_DIR := bin

ifeq ($(OS),Windows_NT)
EXE := .exe
CLEAN := powershell -NoProfile -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $(BIN_DIR)"
else
EXE :=
CLEAN := rm -rf $(BIN_DIR)
endif

.PHONY: build test lint docker clean

build:
	go build -o $(BIN_DIR)/$(BINARY)$(EXE) ./cmd/xnode

test:
	go test ./...

lint:
	go vet ./...

docker:
	docker build -f deploy/Dockerfile -t xnode-agent:local .

clean:
	$(CLEAN)
