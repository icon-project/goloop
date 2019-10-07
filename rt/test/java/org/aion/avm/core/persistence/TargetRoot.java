package org.aion.avm.core.persistence;

import i.IObjectDeserializer;
import i.IObjectSerializer;


public class TargetRoot {
    public static TargetRoot root;
    public final int readIndex;
    public int counter;
    public TargetRoot next;
    
    public TargetRoot() {
        this.readIndex = -1;
    }
    public TargetRoot(Void ignore, int readIndex) {
        this.readIndex = readIndex;
    }
    public void serializeSelf(Class<?> stopBefore, IObjectSerializer serializer) {
        serializer.writeInt(this.counter);
        serializer.writeObject(this.next);
        serializer.automaticallySerializeToRoot((null == stopBefore) ? TargetRoot.class : stopBefore, this);
    }
    
    public void deserializeSelf(Class<?> stopBefore, IObjectDeserializer deserializer) {
        this.counter = deserializer.readInt();
        this.next = (TargetRoot) deserializer.readObject();
        deserializer.automaticallyDeserializeFromRoot((null == stopBefore) ? TargetRoot.class : stopBefore, this);
    }
}
