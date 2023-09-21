#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

BUILD_PLATFORM=$(GOOS)-$(GOARCH)
BUILD_DIR = $(BUILD_ROOT)/build
CROSSBIN_ROOT=$(BUILD_DIR)/bin
ifeq ($(CROSS_COMPILE),TRUE)
BIN_DIR = $(CROSSBIN_ROOT)-$(BUILD_PLATFORM)
else
BIN_DIR = $(BUILD_ROOT)/bin
endif

GOBUILD = go build
GOBUILD_TAGS ?= rocksdb
GOBUILD_ENVS ?= $(GOBUILD_ENVS_$(shell go env GOOS))
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"

GOTEST = go test
GOTEST_FLAGS = -test.short

# Build flags
GL_VERSION ?= $(shell git describe --always --tags --dirty)
GL_TAG ?= latest
BUILD_INFO = $(GOOS)/$(GOARCH) tags($(GOBUILD_TAGS))-$(shell date '+%Y-%m-%d-%H:%M:%S')

#
# Build scripts for command binaries.
#
CMDS = $(patsubst cmd/%,%,$(wildcard cmd/*))
.PHONY: $(CMDS) $(addsuffix -linux,$(CMDS))
define CMD_template
$(BIN_DIR)/$(1) $(1) : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(BIN_DIR)/$(1) $(1) :
	@ \
	rm -f $(BIN_DIR)/$(1) ; \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS) \
	go build $$(GOBUILD_FLAGS) \
	    -o $(BIN_DIR)/$(1) ./cmd/$(1)
endef
$(foreach M,$(CMDS),$(eval $(call CMD_template,$(M))))


# Build flags for each command
gochain_LDFLAGS = -X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += gochain
goloop_LDFLAGS = -X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += goloop

linux : $(addsuffix -linux,$(BUILD_TARGETS))

%-linux:
	@ \
	GOOS=linux GOARCH=amd64 \
	CROSS_COMPILE=TRUE \
	$(MAKE) $(patsubst %-linux,%,$@)

BASE_IMAGE = goloop/base-all:$(GL_TAG)
BASE_PY_IMAGE = goloop/base-py:$(GL_TAG)
BASE_JAVA_IMAGE = goloop/base-java:$(GL_TAG)
BASE_DOCKER_DIR = $(BUILD_DIR)/base

ROCKSDBDEPS_IMAGE = goloop/rocksdb-deps:$(GL_TAG)
GODEPS_IMAGE = goloop/go-deps:$(GL_TAG)
PYDEPS_IMAGE = goloop/py-deps:$(GL_TAG)
JAVADEPS_IMAGE = goloop/java-deps:$(GL_TAG)
BUILDDEPS_IMAGE = goloop/build-deps:$(GL_TAG)
BUILDDEPS_DOCKER_DIR = $(BUILD_DIR)/builddpes

GOCHAIN_IMAGE = goloop/gochain:$(GL_TAG)
GOCHAIN_DOCKER_DIR = $(BUILD_DIR)/gochain

GOLOOP_IMAGE = goloop:$(GL_TAG)
GOLOOP_DOCKER_DIR = $(BUILD_DIR)/goloop

GOLOOP_PY_IMAGE = goloop-py:$(GL_TAG)
GOLOOP_PY_DOCKER_DIR = $(BUILD_DIR)/goloop-py

GOLOOP_JAVA_IMAGE = goloop-java:$(GL_TAG)
GOLOOP_JAVA_DOCKER_DIR = $(BUILD_DIR)/goloop-java

GOLOOP_WORK_DIR = /work
PYEE_DIST_DIR = $(BUILD_ROOT)/build/pyee/dist

$(PYEE_DIST_DIR):
	@ mkdir -p $@

builddeps-%:
	@ \
 	IMAGE_GO_DEPS=$(GODEPS_IMAGE) \
 	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
 	IMAGE_JAVA_DEPS=$(JAVADEPS_IMAGE) \
 	IMAGE_ROCKSDB_DEPS=$(ROCKSDBDEPS_IMAGE) \
	$(BUILD_ROOT)/docker/build-deps/update.sh \
		$(patsubst builddeps-%,%,$@) \
	    goloop/$(patsubst builddeps-%,%,$@)-deps:$(GL_TAG) \
	    $(BUILD_ROOT) $(BUILDDEPS_DOCKER_DIR)

builddeps-build : builddeps-go builddeps-rocksdb

gorun-% : builddeps-build
	@ \
	docker run -t --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR) \
	    -e "CROSS_COMPILE=TRUE" \
	    -e "GOBUILD_TAGS=$(GOBUILD_TAGS)" \
	    -e "GL_VERSION=$(GL_VERSION)" \
	    $(BUILDDEPS_IMAGE) \
	    make $(patsubst gorun-%,%,$@)

pyrun-% : builddeps-py | $(PYEE_DIST_DIR)
	@ \
	docker run -t --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR) \
	    -e "GL_VERSION=$(GL_VERSION)" \
	    $(PYDEPS_IMAGE) \
	    make $(patsubst pyrun-%,%,$@)

pyexec:
	@ \
	echo "[#] Building Python executor" ; \
	cd $(BUILD_ROOT)/pyee ; \
	rm -rf build $(PYEE_DIST_DIR); \
	pip3 install wheel ; \
	python3 setup.py bdist_wheel -d $(PYEE_DIST_DIR) ; \
	rm -rf pyexec.egg-info

javarun-% : builddeps-java
	@ \
	docker run -t --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR)/javaee \
	    $(JAVADEPS_IMAGE) \
	    make $(patsubst javarun-%,%,$@)

base-image-%: builddeps-py builddeps-rocksdb
	@ \
 	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
 	IMAGE_ROCKSDB_DEPS=$(ROCKSDBDEPS_IMAGE) \
	$(BUILD_ROOT)/docker/base/update.sh \
		$(patsubst base-image-%,%,$@) \
	    goloop/base-$(patsubst base-image-%,%,$@):$(GL_TAG) \
	    $(BUILD_ROOT) $(BASE_DOCKER_DIR)-$(patsubst base-image-%,%,$@)

goloop-image: base-image-all pyrun-pyexec gorun-goloop javarun-javaexec
	@ echo "[#] Building image $(GOLOOP_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_DOCKER_DIR)
	@ \
	BIN_DIR=$(CROSSBIN_ROOT)-$$(docker inspect $(BASE_IMAGE) --format "{{.Os}}-{{.Architecture}}") \
	IMAGE_BASE=$(BASE_IMAGE) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop/update.sh $(GOLOOP_IMAGE) $(BUILD_ROOT) $(GOLOOP_DOCKER_DIR)

goloop-py-image: base-image-py pyrun-pyexec gorun-goloop
	@ echo "[#] Building image $(GOLOOP_PY_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_PY_DOCKER_DIR)
	@ \
	BIN_DIR=$(CROSSBIN_ROOT)-$$(docker inspect $(BASE_PY_IMAGE) --format "{{.Os}}-{{.Architecture}}") \
	IMAGE_BASE=$(BASE_PY_IMAGE) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop-py/update.sh \
	    $(GOLOOP_PY_IMAGE) $(BUILD_ROOT) $(GOLOOP_PY_DOCKER_DIR)

goloop-java-image: base-image-java gorun-goloop javarun-javaexec
	@ echo "[#] Building image $(GOLOOP_JAVA_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_JAVA_DOCKER_DIR)
	@ \
	BIN_DIR=$(CROSSBIN_ROOT)-$$(docker inspect $(BASE_JAVA_IMAGE) --format "{{.Os}}-{{.Architecture}}") \
	IMAGE_BASE=$(BASE_JAVA_IMAGE) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop-java/update.sh \
	    $(GOLOOP_JAVA_IMAGE) $(BUILD_ROOT) $(GOLOOP_JAVA_DOCKER_DIR)

gochain-image: base-image-all pyrun-pyexec gorun-gochain javarun-javaexec
	@ echo "[#] Building image $(GOCHAIN_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOCHAIN_DOCKER_DIR)
	@ \
	BIN_DIR=$(CROSSBIN_ROOT)-$$(docker inspect $(BASE_IMAGE) --format "{{.Os}}-{{.Architecture}}") \
	IMAGE_BASE=$(BASE_IMAGE) \
	GOCHAIN_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/gochain/update.sh $(GOCHAIN_IMAGE) $(BUILD_ROOT) $(GOCHAIN_DOCKER_DIR)

.PHONY: test

test :
	$(GOBUILD_ENVS) $(GOTEST) $(GOBUILD_FLAGS) ./... $(GOTEST_FLAGS)

test% : $(BIN_DIR)/gochain
	@ cd testsuite ; ./gradlew $@

test-race :
	$(GOBUILD_ENVS) $(GOTEST) $(GOBUILD_FLAGS) ./... $(GOTEST_FLAGS) -race

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)

-include local.mk
