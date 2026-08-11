.PHONY: build dev demo format

build:
	go build -o radar .

dev: build
	./radar --config config.yaml

demo: build
	./radar --demo

format:
	gofmt -w .
