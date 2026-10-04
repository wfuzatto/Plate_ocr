.PHONY: test build clean

test:
	go test ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/plate-ocr ./cmd/plate-ocr

clean:
	rm -rf bin dist
