/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.util;

import foundation.icon.ee.io.ByteArrayBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.OutputStream;
import java.util.function.Consumer;

public class StringConsumerOutputStream extends OutputStream {
    private static final Logger defLog = LoggerFactory.getLogger(StringConsumerOutputStream.class);

    public static final int TRACE = 0;
    public static final int DEBUG = 1;
    public static final int INFO = 2;
    public static final int WARN = 3;
    public static final int ERROR = 4;

    private ByteArrayBuilder bab = new ByteArrayBuilder();
    private final Consumer<String> consumer;

    public StringConsumerOutputStream(Consumer<String> consumer) {
        this.consumer = consumer;
    }

    @Override
    public void write(int b) {
        this.bab.write(b);
        if (b=='\n') {
            flush();
        }
    }

    @Override
    public void write(byte[] b) {
        write(b, 0, b.length);
    }

    @Override
    public void write(byte[] b, int off, int len) {
        for (int i=off; i<len; i++) {
            write(b[i]);
        }
    }

    @Override
    public void flush() {
        String s = new String(this.bab.array(), 0, this.bab.size());
        this.consumer.accept(s);
        this.bab.resize(0);
    }
}
