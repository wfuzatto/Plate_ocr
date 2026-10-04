.PHONY: test build build-all clean verify

test:
	GOPROXY=off GOSUMDB=off go test ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 GOPROXY=off GOSUMDB=off go build -trimpath -ldflags="-s -w" -o bin/plate-ocr ./cmd/plate-ocr

build-all:
	mkdir -p offline/bin/linux-amd64 offline/bin/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=off GOSUMDB=off go build -trimpath -ldflags="-s -w" -o offline/bin/linux-amd64/plate-ocr ./cmd/plate-ocr
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GOPROXY=off GOSUMDB=off go build -trimpath -ldflags="-s -w" -o offline/bin/linux-arm64/plate-ocr ./cmd/plate-ocr

verify:
	sh scripts/verify-offline.sh

clean:
	rm -rf bin dist offline/bin
