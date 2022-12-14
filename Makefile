VERSION = $(shell GOOS=$(shell go env GOHOSTOS) GOARCH=$(shell go env GOHOSTARCH) \
	go run tools/build-version.go)
SYSTEM = ${GOOS}_${GOARCH}
GOVARS = -X main.Version=$(VERSION)

build:
	go build -trimpath -ldflags "-s -w $(GOVARS)" .

build-dist:
	go build -trimpath -ldflags "-s -w $(GOVARS)" -o dist/bin/eget-$(VERSION)-$(SYSTEM) .

install:
	go install -trimpath -ldflags "-s -w $(GOVARS)" .

fmt:
	gofmt -s -w .

vet:
	go vet

eget:
	go build -trimpath -ldflags "-s -w $(GOVARS)" .

test: eget
	cd test; EGET_CONFIG=eget.toml EGET_BIN= TEST_EGET=../eget go run test_eget.go

eget.1: man/eget.md
	pandoc man/eget.md -s -t man -o eget.1

package: build-dist eget.1
	mkdir -p dist/eget-$(VERSION)-$(SYSTEM)
	cp README.md dist/eget-$(VERSION)-$(SYSTEM)
	cp LICENSE dist/eget-$(VERSION)-$(SYSTEM)
	cp eget.1 dist/eget-$(VERSION)-$(SYSTEM)
	if [ "${GOOS}" = "windows" ]; then\
		cp dist/bin/eget-$(VERSION)-$(SYSTEM) dist/eget-$(VERSION)-$(SYSTEM)/eget.exe;\
		cd dist;\
		zip -r -q -T eget-$(VERSION)-$(SYSTEM).zip eget-$(VERSION)-$(SYSTEM);\
	else\
		cp dist/bin/eget-$(VERSION)-$(SYSTEM) dist/eget-$(VERSION)-$(SYSTEM)/eget;\
		cd dist;\
		tar -czf eget-$(VERSION)-$(SYSTEM).tar.gz eget-$(VERSION)-$(SYSTEM);\
	fi

version:
	echo "package main\n\nvar Version = \"$(VERSION)+src\"" > version.go

clean:
	rm -f test/eget.1 test/fd test/micro test/nvim test/pandoc test/rg.exe
	rm -rf dist

.PHONY: build clean install package version fmt vet test
