package p.avm;

import i.ValueCodec;
import s.java.lang.Object;
import s.java.lang.Byte;

public class PrimitiveBuffer extends Object {
    private byte[] raw;

    public PrimitiveBuffer() {
    }

    public void set(byte[] raw) {
        this.raw = raw;
    }

    public void setByte(byte v) {
    }

    public byte avm_toByte() {
        Byte v = (Byte) ValueCodec.decodeValue(raw);
        return v.getUnderlying();
    }

    public void setShort(byte v) {
    }

    public short avm_toShort() {
        return (short) 0;
    }

    public void setInt(int v) {
    }

    public int avm_toInt() {
        return 0;
    }

    public void setLong(long v) {
    }

    public long avm_toLong() {
        return 0;
    }

    public void setFloat(float v) {
    }

    public float avm_toFloat() {
        return 0;
    }

    public void setDouble(double v) {
    }

    public double avm_toDouble() {
        return 0;
    }

    public void setChar(char v) {
    }

    public char avm_toChar() {
        return 0;
    }

    public void setBoolean(boolean v) {
    }

    public boolean avm_toBoolean() {
        return false;
    }
}
