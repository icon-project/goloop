#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)
BIN_DIR = ./bin
LINUX_BIN_DIR = ./build/linux

GOBUILD = go build
GOBUILD_TAGS ?=
GOBUILD_ENVS_darwin = CGO_ENABLED=0
GOBUILD_ENVS ?= $(GOBUILD_ENVS_$(shell go env GOOS))
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"
GOBUILD_ENVS_LINUX = GOOS=linux GOARCH=amd64

GOTEST = go test
GOTEST_FLAGS = -test.short

# Build flags
GL_VERSION ?= $(shell git describe --always --tags --dirty)
GL_TAG ?= latest
BUILD_INFO = $(shell go env GOOS)/$(shell go env GOARCH) tags($(GOBUILD_TAGS))-$(shell date '+%Y-%m-%d-%H:%M:%S')

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

$(LINUX_BIN_DIR)/$(1) $(1)-linux : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(LINUX_BIN_DIR)/$(1) $(1)-linux :
	@ \
	rm -f $(LINUX_BIN_DIR)/$(1) ; \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS_LINUX) \
	go build $$(GOBUILD_FLAGS) \
	    -o $(LINUX_BIN_DIR)/$(1) ./cmd/$(1)
endef
$(foreach M,$(CMDS),$(eval $(call CMD_template,$(M))))

# Build flags for each command
gochain_LDFLAGS = -X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += gochain
goloop_LDFLAGS = -X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += goloop

linux : $(addsuffix -linux,$(BUILD_TARGETS))

GODEPS_IMAGE = goloop/go-deps:$(GL_TAG)
GODEPS_DOCKER_DIR = $(BUILD_ROOT)/build/godeps

GOCHAIN_IMAGE = goloop/gochain:$(GL_TAG)
GOCHAIN_DOCKER_DIR = $(BUILD_ROOT)/build/gochain

GOLOOP_IMAGE = goloop:$(GL_TAG)
GOLOOP_DOCKER_DIR = $(BUILD_ROOT)/build/goloop

GOLOOP_PY_IMAGE = goloop-py:$(GL_TAG)
GOLOOP_PY_DOCKER_DIR = $(BUILD_ROOT)/build/goloop-py

GOLOOP_JAVA_IMAGE = goloop-java:$(GL_TAG)
GOLOOP_JAVA_DOCKER_DIR = $(BUILD_ROOT)/build/goloop-java

PYDEPS_IMAGE = goloop/py-deps:$(GL_TAG)
PYDEPS_DOCKER_DIR = $(BUILD_ROOT)/build/pydeps

JAVADEPS_IMAGE = goloop/java-deps:$(GL_TAG)
JAVADEPS_DOCKER_DIR = $(BUILD_ROOT)/build/javadeps

GOLOOP_WORK_DIR = /work
PYEE_DIST_DIR = $(BUILD_ROOT)/build/pyee/dist

$(PYEE_DIST_DIR):
	@ mkdir -p $@

godeps-image:
	@ \
	$(BUILD_ROOT)/docker/go-deps/update.sh \
	    $(GODEPS_IMAGE) $(BUILD_ROOT) $(GODEPS_DOCKER_DIR)

gorun-% : godeps-image
	@ \
	docker run -it --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR) \
	    -e "GOBUILD_TAGS=$(GOBUILD_TAGS)" \
	    -e "GL_VERSION=$(GL_VERSION)" \
	    $(GODEPS_IMAGE) \
	    make $(patsubst gorun-%,%,$@)

pydeps-image:
	@ \
	$(BUILD_ROOT)/docker/py-deps/update.sh \
	    $(PYDEPS_IMAGE) $(BUILD_ROOT) $(PYDEPS_DOCKER_DIR)

pyrun-% : pydeps-image | $(PYEE_DIST_DIR)
	@ \
	docker run -it --rm \
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

javadeps-image:
	@ \
	$(BUILD_ROOT)/docker/java-deps/update.sh \
	    $(JAVADEPS_IMAGE) $(BUILD_ROOT) $(JAVADEPS_DOCKER_DIR)

javarun-% : javadeps-image
	@ \
	docker run -it --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR)/javaee \
	    $(JAVADEPS_IMAGE) \
	    make $(patsubst javarun-%,%,$@)

goloop-image: pyrun-pyexec gorun-goloop-linux javarun-javaexec
	@ echo "[#] Building image $(GOLOOP_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_DOCKER_DIR)
	@ \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop/update.sh $(GOLOOP_IMAGE) $(BUILD_ROOT) $(GOLOOP_DOCKER_DIR)

goloop-py-image: pyrun-pyexec gorun-goloop-linux
	@ echo "[#] Building image $(GOLOOP_PY_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_PY_DOCKER_DIR)
	@ \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop-py/update.sh \
	    $(GOLOOP_PY_IMAGE) $(BUILD_ROOT) $(GOLOOP_PY_DOCKER_DIR)

goloop-java-image: gorun-goloop-linux javarun-javaexec
	@ echo "[#] Building image $(GOLOOP_JAVA_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_JAVA_DOCKER_DIR)
	@ \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	GOLOOP_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop-java/update.sh \
	    $(GOLOOP_JAVA_IMAGE) $(BUILD_ROOT) $(GOLOOP_JAVA_DOCKER_DIR)

gochain-image: pyrun-pyexec gorun-gochain-linux javarun-javaexec
	@ echo "[#] Building image $(GOCHAIN_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOCHAIN_DOCKER_DIR)
	@ \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	GOCHAIN_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/gochain/update.sh $(GOCHAIN_IMAGE) $(BUILD_ROOT) $(GOCHAIN_DOCKER_DIR)

.PHONY: test

test :
	$(GOBUILD_ENVS) $(GOTEST) $(GOBUILD_FLAGS) ./... $(GOTEST_FLAGS)

test% : $(BIN_DIR)/gochain
	@ cd testsuite ; ./gradlew $@

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)
