#
# Makefile
# @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
# @author Ievgen Ponomarenko <kikomdev@gmail.com>
#

.PHONY: default help clean build build-static run test test-race vet install uninstall authors deps clean-dist dist docker docker-run docker-tagged image image-test chart-lint chart-template chart-test snap

export GOBIN := $(CURDIR)/bin
export GO111MODULE=on

GO ?= go
GOOS ?= linux
GOARCH ?= amd64
CONTAINER_ENGINE ?= docker
IMAGE_REPOSITORY ?= gobetween-local
IMAGE_TAG ?= udp-dev
CONFIG_FILE ?= $(CURDIR)/config/gobetween-udp.toml
CHART ?= charts/gobetween
HELM ?= helm
NAME := gobetween
VERSION := $(shell cat VERSION)
REVISION := $(shell git rev-parse HEAD 2>/dev/null)
BRANCH := $(shell git symbolic-ref --short HEAD 2>/dev/null)

LDFLAGS := \
  -X main.version=${VERSION} \
  -X main.revision=${REVISION} \
  -X main.branch=${BRANCH}

default: build

help:
	@echo 'build / build-static: local / static Linux executable (GOARCH=amd64 by default)'
	@echo 'test / test-race / vet: check BOTH the root and src modules'
	@echo 'deps: download dependencies for BOTH modules'
	@echo 'image: build IMAGE_REPOSITORY:IMAGE_TAG locally; does not push'
	@echo 'image-test: local Linux Docker + Helm UDP smoke tests (BPF is opt-in)'
	@echo 'chart-lint / chart-template / chart-test: local Helm checks; does not deploy'
	@echo 'Optional integration tests: see README; BPF tests need approved capabilities'

clean:
	@echo Cleaning up...
	@rm bin/* -rf
	@rm dist/* -rf
	@echo Done.

build:
	@echo Building...
	$(GO) build -v -o ./bin/$(NAME) -ldflags '${LDFLAGS}' .
	@echo Done.

build-static:
	@echo Building...
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -v -tags netgo -o ./bin/$(NAME) -ldflags '-s -w ${LDFLAGS}' .
	@echo Done.

run: build
	./bin/$(NAME) -c $(CONFIG_FILE)

test:
	$(GO) test -count=1 ./...
	cd src && $(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...
	cd src && $(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...
	cd src && $(GO) vet ./...

install: build
	install -d ${DESTDIR}/usr/local/bin/
	install -m 755 ./bin/${NAME} ${DESTDIR}/usr/local/bin/${NAME}
	install ./config/${NAME}.toml ${DESTDIR}/etc/${NAME}.toml

uninstall:
	rm -f ${DESTDIR}/usr/local/bin/${NAME}
	rm -f ${DESTDIR}/etc/${NAME}.toml

authors:
	@git log --format='%aN <%aE>' | LC_ALL=C.UTF-8 sort | uniq -c -i | sort -nr | sed "s/^ *[0-9]* //g" > AUTHORS
	@cat AUTHORS

deps: 
	$(GO) mod download
	cd src && $(GO) mod download

clean-dist:
	rm -rf ./dist/${VERSION}

dist:
	@# For linux 386 when building on linux amd64 you'll need 'libc6-dev-i386' package
	@echo Building dist

	@set -e ;\
	for arch in "freebsd   amd64 0    " \
	 		 "linux   386   0      "    \
		     "linux   amd64 0      "    \
		     "linux   arm64 0      "    \
		     "linux   arm   0      "    \
		     "darwin  amd64 0      "    \
		     "windows amd64 0 .exe " ;  \
	do \
		set -- $$arch ; \
		echo "******************* $$1_$$2 ********************" ;\
		distpath="./dist/${VERSION}/$$1_$$2" ;\
		mkdir -p $$distpath ; \
		CGO_ENABLED=$$3 GOOS=$$1 GOARCH=$$2 go build -v -a -tags netgo -o $$distpath/$(NAME)$$4 -ldflags '-s -w --extldflags "-static" ${LDFLAGS}' . ;\
		cp "README.md" "LICENSE" "CHANGELOG.md" "AUTHORS" $$distpath ;\
		mkdir -p $$distpath/config && cp "./config/gobetween.toml" $$distpath/config ;\
		if [ "$$1" = "linux" ]; then \
			cd $$distpath && tar -zcvf ../../${NAME}_${VERSION}_$$1_$$2.tar.gz * && cd - ;\
		else \
			cd $$distpath && zip -r ../../${NAME}_${VERSION}_$$1_$$2.zip . && cd - ;\
		fi \
	done

docker:
	$(MAKE) image

docker-run:
	$(CONTAINER_ENGINE) run --rm --net=host --read-only --cap-drop=ALL --security-opt=no-new-privileges:true \
		-v $(CONFIG_FILE):/etc/gobetween/conf/gobetween.toml:ro \
		$(IMAGE_REPOSITORY):$(IMAGE_TAG)

docker-tagged:
	$(MAKE) image IMAGE_TAG=$(VERSION)

image:
	$(CONTAINER_ENGINE) build --build-arg VERSION=$(VERSION) --build-arg REVISION=$(REVISION) --build-arg BRANCH=$(BRANCH) -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

image-test:
	GOBETWEEN_TEST_IMAGE=$(IMAGE_REPOSITORY):$(IMAGE_TAG) $(GO) test -race -count=1 ./test -run '^TestDockerUDPIntegration$$' -v

chart-lint:
	$(HELM) lint --strict $(CHART)

chart-template:
	$(HELM) template gobetween $(CHART)

chart-test:
	GOBETWEEN_REQUIRE_HELM=1 $(GO) test -count=1 ./test -run '^TestHelmChart$$' -v

snap:
	@echo Building snap for gobetween ${VERSION}
	snapcraft
	@echo Done.
	@echo Install as service: sudo snap install gobetween_0.8.0+snapshot_amd64.snap --dangerous --classic
	@echo Remove: sudo snap remove gobetween
	@echo Config file: /var/snap/gobetween/common/gobetween.toml
	@echo Override start parameters: /var/snap/gobetween/current/gobetween.sh
