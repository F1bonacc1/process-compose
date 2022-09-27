BINARY_NAME=process-compose
RM=rm
VERSION = $(shell git describe --abbrev=0)

LD_FLAGS = "-X main.version=${VERSION} -s -w"

ifeq ($(OS),Windows_NT)
	EXT=.exe
	RM = cmd /C del /Q /F
endif

.PHONY: test run testrace

buildrun: build run

swag:
	~/go/bin/swag init --dir src --output src/docs --parseDependency --parseInternal --parseDepth 1

build:
	go build -o bin/${BINARY_NAME}${EXT} -ldflags="-X main.version=${VERSION}" ./src

compile:
	# Linux
	GOOS=linux GOARCH=386 go build -o bin/${BINARY_NAME}-linux-386 -ldflags=${LD_FLAGS}  ./src
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux-amd64 -ldflags=${LD_FLAGS}  ./src
	GOOS=linux GOARCH=arm64 go build -o bin/${BINARY_NAME}-linux-arm64 -ldflags=${LD_FLAGS}  ./src
	GOOS=linux GOARCH=arm go build -o bin/${BINARY_NAME}-linux-arm -ldflags=${LD_FLAGS}  ./src

	# Windows
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-windows-amd64.exe  -ldflags=${LD_FLAGS} ./src

	# Darwin
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-darwin-amd64 -ldflags=${LD_FLAGS}  ./src
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY_NAME}-darwin-arm64 -ldflags=${LD_FLAGS}  ./src

test:
	go test -cover ./src/...

testrace:
	go test -race ./src/...

coverhtml:
	go test -coverprofile=coverage.out ./src
	go tool cover -html=coverage.out

run:
	PC_DEBUG_MODE=1 ./bin/${BINARY_NAME}${EXT}

clean:
	$(RM) bin/${BINARY_NAME}*
