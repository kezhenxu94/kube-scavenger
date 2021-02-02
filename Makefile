.PHONY: compile build build_all fmt lint test vet

SOURCE_FOLDER := .

BINARY_PATH ?= ./bin/kube-scavenger

GOARCH ?= amd64

default: build

compile:
	CGO_ENABLED=0 go build -i -v -ldflags '-s' -o $(BINARY_PATH) $(SOURCE_FOLDER)/

run:
	go run $(SOURCE_FOLDER)/main.go

build: vet fmt compile
	$(MAKE) compile GOOS=linux GOARCH=amd64

fmt:
	go fmt $(SOURCE_FOLDER)/...

vet:
	go vet $(SOURCE_FOLDER)/...

lint:
	go lint $(SOURCE_FOLDER)/...

test:
	go test $(SOURCE_FOLDER)/...

clean:
	rm -rf bin
