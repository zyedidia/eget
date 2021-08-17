VERSION = $(shell GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) \
	go run tools/build-version.go)
GOVARS = -X main.Version=$(VERSION)

build:
	go build -trimpath -ldflags "-s -w $(GOVARS)" .

install:
	go install -trimpath -ldflags "-s -w $(GOVARS)" .

get.1: man/get.md
	pandoc man/get.md -s -t man -o get.1

package: build get.1
	mkdir get-$(VERSION)
	cp README.md get-$(VERSION)
	cp LICENSE get-$(VERSION)
	cp get.1 get-$(VERSION)
	cp get get-$(VERSION)
	tar -czf get-$(VERSION).tar.gz get-$(VERSION)

clean:
	rm -f get get.1 get-*.tar.gz
	rm -rf get-*/

.PHONY: build clean install package
