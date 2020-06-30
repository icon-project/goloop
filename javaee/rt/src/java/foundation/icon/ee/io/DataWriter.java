package foundation.icon.ee.io;

import java.math.BigInteger;

public interface DataWriter {
    void write(boolean v);
    void write(byte v);
    void write(short v);
    void write(char v);
    void write(int v);
    void write(float v);
    void write(long v);
    void write(double v);
    void write(BigInteger v);
    void write(String v);
    void write(byte[] v);
    void writeNullity(boolean nullity);

    void writeListHeader(int l);
    void writeMapHeader(int l);
    void writeFooter();

    void flush();
    byte[] toByteArray();
    long getTotalWrittenBytes();
}
