package avm;

// charge per byte cost
public interface ObjectWriter {
    void write(boolean v);
    void write(byte v);
    void write(short v);
    void write(char v);
    void write(int v);
    void write(float v);
    void write(long v);
    void write(double v);
    void write(s.java.math.BigInteger v);
    void write(String v);
    void write(byte[] v);
    void write(avm.Address v);
    void write(Object v);
    void writeNullable(Object v);
    void write(Object... v);
    void writeNullable(Object... v);
    void writeNull();

    void beginList(int l);
    void beginNullableList(int l);
    void writeListOf(Object... v);
    void beginMap(int l);
    void beginNullableMap(int l);
    void end();
}
