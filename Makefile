NAME=process-compose
RM=rm
#VERSION = v0.51.0
VERSION = $(shell git describe --abbrev=0)
GIT_REV    ?= $(shell git rev-parse --short HEAD)
DATE       ?= $(shell TZ=UTC0 git show --quiet --date='format-local:%Y-%m-%dT%H:%M:%SZ' --format="%cd")
NUMVER = $(shell echo ${VERSION} | cut -d"v" -f 2)
PKG = github.com/f1bonacc1/${NAME}
SHELL := /bin/bash
PROJ_NAME := Process Compose
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

.PHONY: test run testrace

buildrun: build run

setup:
	go mod tidy

ci: setup build testrace

swag:
	~/go/bin/swag init --dir src --output src/docs --parseDependency --parseInternal --parseDepth 1

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
	PC_DEBUG_MODE=1 ./bin/${NAME}${EXT}

clean:
	$(RM) bin/${NAME}*
release:
	source exports
	goreleaser release --clean --skip-validate
snapshot:
	goreleaser release --snapshot --clean

github-workflows:
	act -W ./.github/workflows/go.yml -j build
	act -W ./.github/workflows/nix.yml -j build
