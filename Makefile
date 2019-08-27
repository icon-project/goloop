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
TARGET_JAR=$(BUILD_DIR)/$(TARGET).jar
TARGET_SO=$(BUILD_DIR)/lib$(TARGET).so

JAVA_HOME ?= /opt/jdk-11.0.2
JAVA=$(JAVA_HOME)/bin/java
JAVAC=$(JAVA_HOME)/bin/javac --release 10
JAR=$(JAVA_HOME)/bin/jar

BUILD_DIR:=./build
LIB_JARS=$(patsubst %:,%,$(shell find lib -name "*.jar" | tr '\n' ':'))

TARGET_CC:=gcc
CC_ARCH_FLAGS = \
      -O3 -fPIC -Wall \
      -I$(JAVA_HOME)/include -I$(JAVA_HOME)/include/linux -I$(BUILD_DIR)/include

JAVA_FILES:=$(shell find src/classes -name "*.java")

OBJ_FILES:=Client.o
vpath %.c  src/native
TARGET_OBJS:=$(patsubst %.o,$(BUILD_DIR)/%.o,$(OBJ_FILES))

TEST_JAVA:=$(shell find test -name "*.java")
TEST_CLASSES:=$(patsubst test/%.java,$(BUILD_DIR)/%.class,$(TEST_JAVA))

all: $(TARGET_JAR) $(TARGET_SO) $(TEST_CLASSES)
clean:
	rm -rf $(BUILD_DIR) $(TARGET_JAR) $(TARGET_SO)

$(TARGET_JAR): $(JAVA_FILES) $(BUILD_DIR)/classes
	$(JAVAC) -cp $(LIB_JARS) -d $(BUILD_DIR)/classes -h $(BUILD_DIR)/include $(JAVA_FILES)
	$(JAR) cfM $@ -C $(BUILD_DIR)/classes .

$(BUILD_DIR)/classes:
	mkdir -p $@

$(BUILD_DIR)/%.o: %.c
	$(TARGET_CC) $(CC_ARCH_FLAGS) -c -o $@ $<

$(TARGET_SO): $(TARGET_OBJS)
	$(TARGET_CC) -shared -o $@ $^

$(BUILD_DIR)/%.class: test/%.java
	$(JAVAC) -cp $(TARGET_JAR):$(LIB_JARS) -d $(BUILD_DIR) $<

run: $(TARGET_JAR) $(TARGET_SO) $(TEST_CLASSES)
	$(JAVA) -cp $(TARGET_JAR):$(LIB_JARS):$(BUILD_DIR) -Djava.library.path=$(BUILD_DIR) \
	    -Dorg.slf4j.simpleLogger.defaultLogLevel=DEBUG \
	    TransactionExecutorTest /tmp/ee.socket uuid1234
