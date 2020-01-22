package pi;

import foundation.icon.ee.util.IObjects;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import s.java.lang.Object;

public abstract class UnmodifiableArrayContainer extends Object {
    IObject[] data;

    UnmodifiableArrayContainer(IObject[] data) {
        this.data = data;
    }

    final int indexOf(IObject o) {
        return IObjects.indexOf(data, o, 0, data.length, 1);
    }

    final int lastIndexOf(IObject o) {
        return IObjects.lastIndexOf(data, o);
    }

    final int indexOf(IObject o, int offset, int step) {
        return IObjects.indexOf(data, o, offset, data.length, step);
    }

    public abstract int avm_size();

    public boolean avm_isEmpty() {
        return avm_size() == 0;
    }

    public void avm_clear() {
        throw new UnsupportedOperationException();
    }

    public IObject[] getData() {
        return data;
    }

    public UnmodifiableArrayContainer(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> cls, IObjectDeserializer deserializer) {
        super.deserializeSelf(UnmodifiableArrayContainer.class, deserializer);

        int length = deserializer.readInt();
        this.data = new IObject[length];
        for (int i = 0; i < length; ++i) {
            this.data[i] = (IObject) deserializer.readObject();
        }
    }

    public void serializeSelf(java.lang.Class<?> cls, IObjectSerializer serializer) {
        super.serializeSelf(UnmodifiableArrayContainer.class, serializer);

        serializer.writeInt(data.length);
        for (IObject e : this.data) {
            serializer.writeObject(e);
        }
    }
}
