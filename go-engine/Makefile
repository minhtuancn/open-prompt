.PHONY: build build-linux build-windows build-darwin test clean

# Build cho host platform
build:
	go build -o bin/go-engine .

# Cross-compile cho từng platform
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/go-engine-linux-amd64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/go-engine-windows-amd64.exe .

build-darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/go-engine-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/go-engine-darwin-amd64 .

test:
	go test ./... -v -race

clean:
	rm -rf bin/
