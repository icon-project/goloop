GOTOOLS = \
	github.com/golang/dep/cmd/dep \
	gopkg.in/alecthomas/gometalinter.v2
PACKAGES=$(shell go list ./...)
INCLUDE = -I=. -I=${GOPATH}/src 
BUILD_TAGS =
BUILD_FLAGS =

init: get_tools get_vendor_deps

all: check build test install

check: ensure_deps

build:
	@echo "--> Building"
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' $(PACKAGES)

install:
	@echo "--> Installing"
	CGO_ENABLED=0 go install $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' $(PACKAGES)

test:
	@echo "--> Running go test"
	@go test $(PACKAGES)

ensure_deps:
	@rm -rf vendor/
	@echo "--> Running dep"
	@dep ensure

get_tools:
	@echo "--> Installing tools"
	go get -u -v $(GOTOOLS)
	@gometalinter.v2 --install

update_tools:
	@echo "--> Updating tools"
	@go get -u $(GOTOOLS)

get_vendor_deps:
	@rm -rf vendor/
	@mkdir vendor/
	@echo "--> Running dep"
	@dep ensure -vendor-only
