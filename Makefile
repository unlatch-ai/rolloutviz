.PHONY: build check clean format test

build:
	mkdir -p bin
	go build -o bin/rolloutviz ./cmd/rolloutviz

test:
	go test ./...

format:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

check:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"
	go vet ./...
	go test ./...

clean:
	rm -rf bin
