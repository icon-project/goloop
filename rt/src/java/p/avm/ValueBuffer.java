package p.avm;

import s.java.lang.Object;

public class ValueBuffer extends Object {
    private byte[] raw;

    public ValueBuffer() {
    }

    public void set(byte[] raw) {
        this.raw = raw;
    }

    public void setByte(byte v) {
    }

    public byte avm_toByte() {
        return (byte) 0;
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
