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
    void skip(int count);

    void readListHeader();
    void readMapHeader();
    boolean hasNext();
    void readFooter();
    long getTotalReadBytes();
}
