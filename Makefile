NAME=process-compose
RM=rm
VERSION = $(shell git describe --abbrev=0)
GIT_REV    ?= $(shell git rev-parse --short HEAD)
DATE       ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
NUMVER = $(shell echo ${VERSION} | cut -d"v" -f 2)
PKG = github.com/f1bonacc1/${NAME}
SHELL := /bin/bash
LD_FLAGS := -ldflags="-X ${PKG}/src/config.Version=${VERSION} \
            -X ${PKG}/src/config.CheckForUpdates=true \
            -X ${PKG}/src/config.Commit=${GIT_REV} \
            -X ${PKG}/src/config.Date=${DATE} \
            -s -w"
ifeq ($(OS),Windows_NT)
	EXT=.exe
	RM = cmd /C del /Q /F
endif

.PHONY: test run testrace

buildrun: build run

swag:
	~/go/bin/swag init --dir src --output src/docs --parseDependency --parseInternal --parseDepth 1

build:
	go build -o bin/${NAME}${EXT} ${LD_FLAGS} ./src

build-nix:
	nix build .

nixver:
	sed -i 's/version = ".*"/version = "${NUMVER}"/' default.nix

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
	go test -coverprofile=coverage.out ./src
	go tool cover -html=coverage.out

run:
	PC_DEBUG_MODE=1 ./bin/${NAME}${EXT}

clean:
	$(RM) bin/${NAME}*
release:
	source exports
	goreleaser release --rm-dist --skip-validate
snapshot:
	goreleaser release --snapshot --rm-dist
