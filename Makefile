VERSION := $(shell sh -c 'git describe --always --tags')
BRANCH := $(shell sh -c 'git rev-parse --abbrev-ref HEAD')
COMMIT := $(shell sh -c 'git rev-parse HEAD')
ifdef GOBIN
PATH := $(GOBIN):$(PATH)
else
PATH := $(subst :,/bin:,$(GOPATH))/bin:$(PATH)
endif

# Standard lrconf build
default: prepare build

# Windows build
windows: prepare-windows build-windows

# Only run the build (no dependency grabbing)
build:
	go build -o bin/lrconf-agent  -ldflags \
		"-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)" \
		./pkg/agent/

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/lrconf-agent.exe -ldflags \
		"-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)" \
		./pkg/agent/

# run package script
package:
	./scripts/build.py --package --version="$(VERSION)" --platform=linux --arch=all --upload

# Get dependencies and use gdm to checkout changesets
prepare:
	go get github.com/sparrc/gdm
	gdm restore

# Use the windows godeps file to prepare dependencies
prepare-windows:
	go get github.com/sparrc/gdm
	gdm restore
	gdm restore -f Godeps_windows

# Run "short" unit tests
test-short: vet
	go test -short ./...

vet:
	go vet ./...

.PHONY: test test-short vet build default
