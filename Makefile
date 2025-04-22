NAME=process-compose
RM=rm
#VERSION = v0.51.0
VERSION = $(shell git describe --abbrev=0)
GIT_REV    ?= $(shell git rev-parse --short HEAD)
DATE       ?= $(shell TZ=UTC0 git show --quiet --date='format-local:%Y-%m-%dT%H:%M:%SZ' --format="%cd")
NUMVER = $(shell echo ${VERSION} | cut -d"v" -f 2)
PKG = github.com/f1bonacc1/${NAME}
SHELL := /usr/bin/env bash
PROJ_NAME := Process Compose
DOCS_DIR  := www/docs/cli
LD_FLAGS := -ldflags="-X ${PKG}/src/config.Version=${VERSION} \
            -X ${PKG}/src/config.CheckForUpdates=true \
            -X ${PKG}/src/config.Commit=${GIT_REV} \
            -X ${PKG}/src/config.Date=${DATE} \
            -X '${PKG}/src/config.ProjectName=${PROJ_NAME} 🔥' \
            -X '${PKG}/src/config.RemoteProjectName=${PROJ_NAME} ⚡' \
            -s -w"
ifeq ($(OS),Windows_NT)
	EXT=.exe
	RM = cmd /C del /Q /F
endif

.PHONY: test run testrace docs schema

buildrun: build run

setup:
	go mod download

ci: setup build testrace

swag: swag2op ## Generate docs from swagger attributes in the code
	./bin/swag2op init --dir src --output src/docs -g api/pc_api.go --openapiOutputDir src/docs --parseDependency --parseInternal

build:
	CGO_ENABLED=0 go build -o bin/${NAME}${EXT} ${LD_FLAGS} ./src

build-nix:
	nix build .

nixver:
	sed -i 's/version = ".*"/version = "${NUMVER}"/' default.nix

build-pi:
	GOOS=linux GOARCH=arm go build ${LD_FLAGS} -o bin/${NAME}-linux-arm  ./src

compile:
	for arch in amd64 386 arm64 arm; do \
		GOOS=linux GOARCH=$$arch go build ${LD_FLAGS} -o bin/${NAME}-linux-$$arch  ./src ; \
	done;

	for arch in amd64 arm64; do \
		GOOS=darwin GOARCH=$$arch go build ${LD_FLAGS} -o bin/${NAME}-darwin-$$arch  ./src ; \
	done;

	for arch in amd64 arm64; do \
		GOOS=windows GOARCH=$$arch go build ${LD_FLAGS} -o bin/${NAME}-windows-$$arch.exe  ./src ; \
	done;

test:
	go test -cover ./src/...

testrace:
	go test -race ./src/...

coverhtml:
	go test -coverprofile=coverage.out ./src/...
	go tool cover -html=coverage.out

run:
	PC_DEBUG_MODE=1 ./bin/${NAME}${EXT} -e .env

clean:
	$(RM) bin/${NAME}*
release:
	source exports
	goreleaser release --clean --skip validate
snapshot:
	goreleaser release --snapshot --clean

github-workflows:
	act -W ./.github/workflows/go.yml -j build
	act -W ./.github/workflows/nix.yml -j build

docs: build
	./bin/process-compose docs ${DOCS_DIR}
	for f in ${DOCS_DIR}/*.md ; do sed -i 's/${USER}/<user>/g; s|${TMPDIR}|/tmp/|g; s/process-compose-[0-9]\+.sock/process-compose-<pid>.sock/g' $$f ; done

schema:
	./bin/process-compose schema ./schemas/process-compose-schema.json

lint: golangci-lint
	./bin/golangci-lint run --show-stats -c .golangci.yaml

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
SWAG2OP_GEN ?= $(LOCALBIN)/swag2op
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

.PHONY: swag2op
swag2op: $(SWAG2OP_GEN) ## Download swag2op locally if necessary.
$(SWAG2OP_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/swag2op || \
	GOBIN=$(LOCALBIN) go install github.com/zxmfke/swagger2openapi3/cmd/swag2op@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
