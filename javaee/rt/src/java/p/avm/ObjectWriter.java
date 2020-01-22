package p.avm;

import a.ByteArray;
import i.IObject;
import i.IObjectArray;

public interface ObjectWriter {
    void avm_write(boolean v);
    void avm_write(byte v);
    void avm_write(short v);
    void avm_write(char v);
    void avm_write(int v);
    void avm_write(float v);
    void avm_write(long v);
    void avm_write(double v);
    void avm_write(s.java.math.BigInteger v);
    void avm_write(s.java.lang.String v);
    void avm_write(ByteArray v);
    void avm_write(Address v);
    void avm_write(IObject v);
    void avm_writeNullable(IObject v);
    void avm_write(IObjectArray v);
    void avm_writeNullable(IObjectArray v);
    void avm_writeNull();

    void avm_beginList(int l);
    void avm_beginNullableList(int l);
    void avm_writeListOf(IObjectArray v);
    void avm_beginMap(int l);
    void avm_beginNullableMap(int l);
    void avm_end();
}
