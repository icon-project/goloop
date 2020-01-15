#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)
BIN_DIR = ./bin
LINUX_BIN_DIR = ./build/linux

GOBUILD = go build
GOBUILD_TAGS =
GOBUILD_ENVS = CGO_ENABLED=0
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"
GOBUILD_ENVS_LINUX = $(GOBUILD_ENVS) GOOS=linux GOARCH=amd64

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

PYDEPS_IMAGE = goloop/py-deps:$(GL_TAG)
PYDEPS_DOCKER_DIR = $(BUILD_ROOT)/build/pydeps

JAVADEPS_IMAGE = goloop/java-deps:$(GL_TAG)
JAVADEPS_DOCKER_DIR = $(BUILD_ROOT)/build/javadeps

PYJAVADEPS_IMAGE = goloop/pyjava-deps:$(GL_TAG)
PYJAVADEPS_DOCKER_DIR = $(BUILD_ROOT)/build/pyjavadeps

GOLOOP_WORK_DIR = /work
PYEE_DIST_DIR = $(BUILD_ROOT)/build/pyee/dist

godeps-image:
	@ \
	$(BUILD_ROOT)/docker/go-deps/update.sh \
	    $(GODEPS_IMAGE) $(BUILD_ROOT) $(GODEPS_DOCKER_DIR)

gorun-% : godeps-image
	@ \
	docker run -it --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR) \
	    $(GODEPS_IMAGE) \
	    make "GL_VERSION=$(GL_VERSION)" $(patsubst gorun-%,%,$@)

pydeps-image:
	@ \
	$(BUILD_ROOT)/docker/py-deps/update.sh \
	    $(PYDEPS_IMAGE) $(BUILD_ROOT) $(PYDEPS_DOCKER_DIR)

pyrun-% : pydeps-image
	@ \
	docker run -it --rm \
	    -v $(BUILD_ROOT):$(GOLOOP_WORK_DIR) \
	    -w $(GOLOOP_WORK_DIR) \
	    $(PYDEPS_IMAGE) \
	    make "GL_VERSION=$(GL_VERSION)" $(patsubst pyrun-%,%,$@)

pyexec:
	@ \
	cd $(BUILD_ROOT)/pyee ; \
	rm -rf build ; \
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
	    -w $(GOLOOP_WORK_DIR) \
	    $(JAVADEPS_IMAGE) \
	    make "GL_VERSION=$(GL_VERSION)" $(patsubst javarun-%,%,$@)

javaexec:
	@ \
	cd $(BUILD_ROOT)/javaee ; \
	./gradlew app:exectest:distZip

pyjavadeps-image:
	@ \
	$(BUILD_ROOT)/docker/pyjava-deps/update.sh \
	    $(PYJAVADEPS_IMAGE) $(BUILD_ROOT) $(PYJAVADEPS_DOCKER_DIR)

goloop-image: pyrun-pyexec gorun-goloop-linux javarun-javaexec pyjavadeps-image
	@ echo "[#] Building image $(GOLOOP_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOLOOP_DOCKER_DIR)
	@ mkdir -p $(GOLOOP_DOCKER_DIR)/dist/pyee
	@ mkdir -p $(GOLOOP_DOCKER_DIR)/dist/bin
	@ cp $(BUILD_ROOT)/docker/goloop/* $(GOLOOP_DOCKER_DIR)
	@ cp $(PYEE_DIST_DIR)/*.whl $(GOLOOP_DOCKER_DIR)/dist/pyee
	@ cp $(LINUX_BIN_DIR)/goloop $(GOLOOP_DOCKER_DIR)/dist/bin
	@ cp $(BUILD_ROOT)/javaee/app/exectest/build/distributions/exectest.zip $(GOLOOP_DOCKER_DIR)/dist
	@ docker build -t $(GOLOOP_IMAGE) \
	    --build-arg TAG_PYJAVA_DEPS=$(GL_TAG) \
	    --build-arg GOLOOP_VERSION=$(GL_VERSION) \
	    $(GOLOOP_DOCKER_DIR)

gochain-image: pyrun-pyexec gorun-gochain-linux javarun-javaexec pyjavadeps-image
	@ echo "[#] Building image $(GOCHAIN_IMAGE) for $(GL_VERSION)"
	@ rm -rf $(GOCHAIN_DOCKER_DIR)
	@ mkdir -p $(GOCHAIN_DOCKER_DIR)/dist
	@ cp $(BUILD_ROOT)/docker/gochain/* $(GOCHAIN_DOCKER_DIR)
	@ cp $(PYEE_DIST_DIR)/*.whl $(GOCHAIN_DOCKER_DIR)/dist
	@ cp $(LINUX_BIN_DIR)/gochain $(GOCHAIN_DOCKER_DIR)/dist
	@ cp $(BUILD_ROOT)/javaee/app/exectest/build/distributions/exectest.zip $(GOCHAIN_DOCKER_DIR)/dist
	@ docker build -t $(GOCHAIN_IMAGE) \
	    --build-arg TAG_PYJAVA_DEPS=$(GL_TAG) \
	    --build-arg GOCHAIN_VERSION=$(GL_VERSION) \
	    $(GOCHAIN_DOCKER_DIR)

.PHONY: test

test :
	$(GOBUILD_ENVS) $(GOTEST) $(GOBUILD_FLAGS) ./... $(GOTEST_FLAGS)

test% : $(BIN_DIR)/gochain
	@ cd testsuite ; ./gradlew $@

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)
