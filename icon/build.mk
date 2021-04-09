#
#  Makefile for ICON2
#

LCIMPORT_IMAGE = goloop/lcimport:$(GL_TAG)
LCIMPORT_DOCKER_DIR = $(BUILD_ROOT)/build/lcimport
lcimport_LDFLAGS = -X 'main.version=$(GL_VERSION)'

GOCHAIN_ICON_IMAGE = goloop/gochain-icon:$(GL_TAG)
GOCHAIN_ICON_DOCKER_DIR = $(BUILD_ROOT)/build/gochain-icon

GOLOOP_ICON_IMAGE = goloop-icon:$(GL_TAG)
GOLOOP_ICON_DOCKER_DIR = $(BUILD_ROOT)/build/goloop-icon

ICONEE_DIST_DIR = $(BUILD_ROOT)/build/iconee/dist

$(ICONEE_DIST_DIR):
	@ mkdir -p $@

iconexec:
	@ \
	echo "[#] Building ICON executor" ; \
	cd $(BUILD_ROOT)/iconee ; \
	rm -rf build $(ICONEE_DIST_DIR); \
	pip3 install wheel ; \
	python3 setup.py bdist_wheel -d $(ICONEE_DIST_DIR) ; \
	rm -rf iconee.egg-info

lcimport-image: pyrun-iconexec gorun-lcimport-linux
	@ echo "[#] Building lcimport for $(GL_VERSION)"
	@ \
	rm -rf $(LCIMPORT_DOCKER_DIR); \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	LCIMPORT_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/lcimport/update.sh $(LCIMPORT_IMAGE) $(BUILD_ROOT) $(LCIMPORT_DOCKER_DIR)

gochain-icon-image: pyrun-iconexec gorun-gochain-linux javarun-javaexec
	@ echo "[#] Building image $(GOCHAIN_ICON_IMAGE) for $(GL_VERSION)"
	@ \
	rm -rf $(GOCHAIN_ICON_DOCKER_DIR); \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	GOCHAIN_ICON_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/gochain-icon/update.sh $(GOCHAIN_ICON_IMAGE) $(BUILD_ROOT) $(GOCHAIN_ICON_DOCKER_DIR)

goloop-icon-image: pyrun-iconexec gorun-goloop-linux javarun-javaexec
	@ echo "[#] Building image $(GOLOOP_ICON_IMAGE) for $(GL_VERSION)"
	@ \
	rm -rf $(GOLOOP_ICON_DOCKER_DIR); \
	BIN_DIR=$(abspath $(LINUX_BIN_DIR)) \
	IMAGE_PY_DEPS=$(PYDEPS_IMAGE) \
	GOLOOP_ICON_VERSION=$(GL_VERSION) \
	GOBUILD_TAGS="$(GOBUILD_TAGS)" \
	$(BUILD_ROOT)/docker/goloop-icon/update.sh $(GOLOOP_ICON_IMAGE) $(BUILD_ROOT) $(GOLOOP_ICON_DOCKER_DIR)
