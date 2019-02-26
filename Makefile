#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BIN_DIR = ./bin
LINUX_BIN_DIR = ./linux

GOBUILD = go build
GOBUILD_TAGS =
GOBUILD_ENVS = CGO_ENABLED=0
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"
GOBUILD_ENVS_LINUX = $(GOBUILD_ENVS) GOOS=linux GOARCH=amd64 

# Build flags
VERSION ?= $(shell git describe --always --tags --dirty)
BUILD_INFO = tags($(GOBUILD_TAGS))-$(shell date '+%Y-%m-%d-%H:%M:%S')

#
# Build scripts for command binaries.
#
CMDS = $(patsubst cmd/%,%,$(wildcard cmd/*))
.PHONY: $(CMDS)
define CMD_template
$(BIN_DIR)/$(1) : $(1)
$(1) : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(1) : | vendor
	@ \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS) \
	go build $$(GOBUILD_FLAGS) \
	    -o $(BIN_DIR)/$(1) ./cmd/$(1)

$(LINUX_BIN_DIR)/$(1) : $(1)-linux
$(1)-linux : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(1)-linux : | vendor
	@ \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS_LINUX) \
	go build $$(GOBUILD_FLAGS) \
	    -o $(LINUX_BIN_DIR)/$(1) ./cmd/$(1)
endef
$(foreach M,$(CMDS),$(eval $(call CMD_template,$(M))))

# Build flags for each command
gochain_LDFLAGS = -X 'main.version=$(VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += gochain

vendor :
	@ \
	$(MAKE) ensure
ensure :
	@ \
	echo "[#] dep ensure"
	dep ensure

linux : $(addsuffix -linux,$(BUILD_TARGETS))

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)
