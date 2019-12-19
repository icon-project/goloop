package org.aion.avm.core.persistence;

import java.nio.ByteBuffer;

public class TestObjectSerializer {
    private final ByteBuffer buffer;

    public TestObjectSerializer(ByteBuffer buffer) {
        this.buffer = buffer;
    }

    public void writeInt(int value) {
        this.buffer.putInt(value);
    }

    public void writeIntArray(int[] value) {
        for (int i = 0; i < value.length; i++) {
            this.buffer.putInt(value[i]);
        }
    }
}
