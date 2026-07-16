.PHONY: build check clean dev format test web-build web-install

build: web-build
	mkdir -p bin
	go build -o bin/rlviz ./cmd/rolloutviz
	cp bin/rlviz bin/rolloutviz

web-install:
	npm --prefix web ci

web-build:
	npm --prefix web run build

test:
	go test ./...
	npm --prefix web test

format:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

check:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"
	go vet ./...
	go test ./...
	npm --prefix web test
	npm --prefix web run build

dev:
	npm --prefix web run dev

clean:
	rm -rf bin web/dist
