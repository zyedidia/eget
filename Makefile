VERSION = $(shell GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) \
	go run tools/build-version.go)
SYSTEM = ${GOOS}_${GOARCH}
GOVARS = -X main.Version=$(VERSION)

build:
	go build -trimpath -ldflags "-s -w $(GOVARS)" .

install:
	go install -trimpath -ldflags "-s -w $(GOVARS)" .

get.1: man/get.md
	pandoc man/get.md -s -t man -o get.1

package: build get.1
	mkdir get-$(VERSION)-$(SYSTEM)
	cp README.md get-$(VERSION)-$(SYSTEM)
	cp LICENSE get-$(VERSION)-$(SYSTEM)
	cp get.1 get-$(VERSION)-$(SYSTEM)
	cp get get-$(VERSION)-$(SYSTEM)
	if [ ${GOOS} = "windows" ]; then\
		zip -r -q -T get-$(VERSION)-$(SYSTEM).zip get-$(VERSION)-$(SYSTEM);\
	else\
		tar -czf get-$(VERSION)-$(SYSTEM).tar.gz get-$(VERSION)-$(SYSTEM);\
	fi

clean:
	rm -f get get.exe get.1 get-*.tar.gz
	rm -rf get-*/

.PHONY: build clean install package
