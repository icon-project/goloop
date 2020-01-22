package avm;

import java.math.BigInteger;

public interface ObjectReader {
    boolean readBoolean();
    byte readByte();
    short readShort();
    char readChar();
    int readInt();
    float readFloat();
    long readLong();
    double readDouble();
    BigInteger readBigInteger();
    String readString();
    byte[] readByteArray();
    Address readAddress();
    <T> T read(Class<T> c);
    <T> T readOrDefault(Class<T> c, T def);
    <T> T readNullable(Class<T> c);
    <T> T readNullableOrDefault(Class<T> c, T def);

    // returns length of list or -1 if unknown
    void beginList();
    void beginNullableList();
    boolean tryBeginNullableList();
    void beginMap();
    void beginNullableMap();
    boolean tryBeginNullableMap();
    boolean hasNext();
    void end();
    boolean tryReadNull();

    void skip();
    void skip(int count);
}
