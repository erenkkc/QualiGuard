.PHONY: build build-server build-all build-extension test scan scan-sarif tidy

build:
	go build -o bin/qg ./cmd/qg

build-server:
	go build -o bin/qg-server.exe ./cmd/qg-server

build-all: build build-server

build-extension:
	cd extension/qualiguard && npm install && npm run compile

test:
	go test ./...
tidy:
	go mod tidy

scan:
	go run ./cmd/qg scan --config qualiguard.yaml --verbose

scan-sarif:
	go run ./cmd/qg scan --config qualiguard.yaml --format sarif --output report.sarif
