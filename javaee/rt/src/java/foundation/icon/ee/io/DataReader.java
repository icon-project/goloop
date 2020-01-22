package foundation.icon.ee.io;

import java.math.BigInteger;

public interface DataReader {
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
    boolean readNullity();
    boolean tryReadNull();
    void skip(int count);

    // returns length of list or -1 if unknown
    void readListHeader();
    void readMapHeader();
    boolean hasNext();
    void readFooter();
}
