package i;


/**
 * Passed to serializeSelf() so that the receiver can abstractly serialize itself.
 * Note that there is no identification of data elements, other than the order they are written.
 */
public interface IObjectSerializer {
    void writeBoolean(boolean value);
    void writeByte(byte value);
    void writeShort(short value);
    void writeChar(char value);
    void writeInt(int value);
    void writeFloat(float value);
    void writeLong(long value);
    void writeDouble(double value);
    void writeByteArray(byte[] value);
    void writeObject(Object value);
    void writeClassName(String internalClassName);
    void automaticallySerializeToRoot(Class<?> rootClass, Object instance);
}
