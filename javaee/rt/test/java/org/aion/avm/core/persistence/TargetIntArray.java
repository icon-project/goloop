package org.aion.avm.core.persistence;

import i.IObjectDeserializer;
import i.IObjectSerializer;


public final class TargetIntArray extends TargetRoot {
    public int[] array;
    public TargetIntArray(int size) {
        this.array = new int[size];
    }
    public TargetIntArray(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
    
    public void serializeSelf(Class<?> stopBefore, IObjectSerializer serializer) {
        super.serializeSelf(TargetIntArray.class, serializer);
        serializer.writeInt(this.array.length);
        for (int elt : this.array) {
            serializer.writeInt(elt);
        }
    }
    
    public void deserializeSelf(Class<?> stopBefore, IObjectDeserializer deserializer) {
        super.deserializeSelf(TargetIntArray.class, deserializer);
        int size = deserializer.readInt();
        this.array = new int[size];
        for (int i = 0; i < size; ++i) {
            this.array[i] = deserializer.readInt();
        }
    }

    public void serializeForTest(Class<?> stopBefore, TestObjectSerializer serializer) {
        serializer.writeInt(this.array.length);
        serializer.writeIntArray(this.array);
    }
}
