package p.avm;

import a.ByteArray;
import i.IObject;

// charges 0 cost
public interface ObjectReader {
    // UnsupportedOperationException : invalid or unsupported format
    // IllegalStateException : programming error or unexpected stream
    // NoSuchElementException : unexpected end of container
    boolean avm_readBoolean();
    byte avm_readByte();
    short avm_readShort();
    char avm_readChar();
    int avm_readInt();
    float avm_readFloat();
    long avm_readLong();
    double avm_readDouble();
    s.java.math.BigInteger avm_readBigInteger();
    s.java.lang.String avm_readString();
    ByteArray avm_readByteArray();
    Address avm_readAddress();
    <T extends IObject> T avm_read(s.java.lang.Class<T> c);
    <T extends IObject> T avm_readOrDefault(s.java.lang.Class<T> c, T def);
    <T extends IObject> T avm_readNullable(s.java.lang.Class<T> c);
    <T extends IObject> T avm_readNullableOrDefault(s.java.lang.Class<T> c, T def);

    // returns length of list or -1 if unknown
    void avm_beginList();
    void avm_beginNullableList();
    boolean avm_tryBeginNullableList();
    void avm_beginMap();
    void avm_beginNullableMap();
    boolean avm_tryBeginNullableMap();
    boolean avm_hasNext();
    void avm_end();
    boolean avm_tryReadNull();

    void avm_skip();
    void avm_skip(int count);

}
