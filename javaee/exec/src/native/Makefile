# Copyright 2019 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TARGET:=client
TARGET_SO=$(BUILD_DIR)/lib$(TARGET).$(LIB_SUFFIX)

ifneq (, $(findstring Darwin, $(shell uname -s)))
  JAVA_HOME ?= $(shell /usr/libexec/java_home -v 11.0)
  OS_DIR = darwin
  LIB_SUFFIX = dylib
else
  JAVA_HOME ?= /opt/jdk-11.0.2
  OS_DIR = linux
  LIB_SUFFIX = so
endif

BUILD_DIR:=../../build/native

TARGET_CC:=gcc
CC_ARCH_FLAGS = \
      -O3 -fPIC -Wall \
      -I$(JAVA_HOME)/include -I$(JAVA_HOME)/include/$(OS_DIR)

OBJ_FILES:=Client.o
vpath %.c  .
TARGET_OBJS:=$(patsubst %.o,$(BUILD_DIR)/%.o,$(OBJ_FILES))

all: $(TARGET_SO)
clean:
	rm -rf $(BUILD_DIR)

$(BUILD_DIR):
	mkdir -p $@

$(BUILD_DIR)/%.o: %.c
	$(TARGET_CC) $(CC_ARCH_FLAGS) -c -o $@ $<

$(TARGET_OBJS): | $(BUILD_DIR)

$(TARGET_SO): $(TARGET_OBJS)
	$(TARGET_CC) -shared -o $@ $^
