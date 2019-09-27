package i;


/**
 * Passed to deserializeSelf() so that the receiver can abstractly deserialize itself.
 * Note that there is no identification of data elements, other than the order they are read.
 */
public interface IObjectDeserializer {
    boolean readBoolean();
    byte readByte();
    short readShort();
    char readChar();
    int readInt();
    float readFloat();
    long readLong();
    double readDouble();
    void readByteArray(byte[] result);
    Object readObject();
    String readClassName();
    void automaticallyDeserializeFromRoot(Class<?> rootClass, Object instance);
}
