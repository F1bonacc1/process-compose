BINARY_NAME=process-compose
EXT=''
RM=rm
ifeq ($(OS),Windows_NT)
	EXT=.exe
	RM = cmd /C del /Q /F
endif

.PHONY: test run

buildrun: build run

build:
	go build -o bin/${BINARY_NAME}${EXT} ./src
compile:
	# Linux
	GOOS=linux GOARCH=386 go build -o bin/${BINARY_NAME}-linux-386 ./src
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux-amd64 ./src
	GOOS=linux GOARCH=arm64 go build -o bin/${BINARY_NAME}-linux-arm64 ./src
	GOOS=linux GOARCH=arm go build -o bin/${BINARY_NAME}-linux-arm ./src

	# Windows
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-windows-amd64.exe ./src

test:
	go test -cover ./src
coverhtml:
	go test -coverprofile=coverage.out ./src
	go tool cover -html=coverage.out

run:
	./bin/${BINARY_NAME}${EXT}

clean:
	$(RM) bin/${BINARY_NAME}*
