.PHONY: all build test vet lint fmt fmt-check coverage vuln clean install

all: build

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	staticcheck ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi

coverage:
	go test -race -coverprofile=coverage.txt ./...

vuln:
	govulncheck ./...

clean:
	rm -f coverage.txt

install:
	go install ./cmd/geoctl