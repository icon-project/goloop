package org.aion.avm.core.persistence;

import i.IObjectDeserializer;
import i.IObjectSerializer;


public final class TargetArray extends TargetRoot {
    public Object[] array;
    public TargetArray(int size) {
        this.array = new Object[size];
    }
    public TargetArray(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
    
    public void serializeSelf(Class<?> stopBefore, IObjectSerializer serializer) {
        super.serializeSelf(TargetArray.class, serializer);
        serializer.writeInt(this.array.length);
        for (Object elt : this.array) {
            serializer.writeObject(elt);
        }
    }
    
    public void deserializeSelf(Class<?> stopBefore, IObjectDeserializer deserializer) {
        super.deserializeSelf(TargetArray.class, deserializer);
        int size = deserializer.readInt();
        this.array = new Object[size];
        for (int i = 0; i < size; ++i) {
            this.array[i] = deserializer.readObject();
        }
    }
}
