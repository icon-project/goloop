package p.avm;

import a.ByteArray;
import s.java.lang.Object;
import s.java.lang.String;

import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;

public class ValueBuffer extends Object implements Value {
    private byte[] raw;

    public ValueBuffer() {
        raw = new byte[0];
    }

    public ValueBuffer(byte v) {
        set(v);
    }

    public ValueBuffer(short v) {
        set(v);
    }

    public ValueBuffer(int v) {
        set(v);
    }

    public ValueBuffer(long v) {
        set(v);
    }

    public ValueBuffer(float v) {
        set(v);
    }

    public ValueBuffer(double v) {
        set(v);
    }

    public ValueBuffer(char v) {
        set(v);
    }

    public ValueBuffer(boolean v) {
        set(v);
    }

    public ValueBuffer(s.java.math.BigInteger v) {
        set(v);
    }

    public ValueBuffer(Address v) {
        set(v);
    }

    public ValueBuffer(String v) {
        set(v);
    }

    public ValueBuffer(byte[] v) {
        set(v);
    }

    public void set(byte[] raw) {
        this.raw = raw;
    }

    public void avm_set(byte v) {
        set(v);
    }

    public byte avm_asByte() {
        return (new BigInteger(raw)).byteValue();
    }

    public void avm_set(short v) {
        set(v);
    }

    public short avm_asShort() {
        return (new BigInteger(raw)).shortValue();
    }

    public void avm_set(int v) {
        set(v);
    }

    public int avm_asInt() {
        return asInt();
    }

    public int asInt() {
        return (new BigInteger(raw)).intValue();
    }

    public void avm_set(long v) {
        set(v);
    }

    public long avm_asLong() {
        return (new BigInteger(raw)).longValue();
    }

    public void avm_set(float v) {
        set(v);
    }

    public float avm_asFloat() {
        return ByteBuffer.wrap(raw).getFloat();
    }

    public void avm_set(double v) {
        set(v);
    }

    public double avm_asDouble() {
        return ByteBuffer.wrap(raw).getDouble();
    }

    public void avm_set(char v) {
        set(v);
    }

    public char avm_asChar() {
        return (char) (new BigInteger(raw)).intValue();
    }

    public void avm_set(boolean v) {
        set(v);
    }

    public boolean avm_asBoolean() {
        return (new BigInteger(raw)).intValue() != 0;
    }

    public void avm_set(s.java.math.BigInteger v) {
        set(v);
    }

    public s.java.math.BigInteger avm_asBigInteger() {
        return new s.java.math.BigInteger(new BigInteger(raw));
    }

    public void avm_set(Address v) {
        set(v);
    }

    public Address avm_asAddress() {
        return new Address(raw);
    }

    public void avm_set(String v) {
        set(v);
    }

    public String avm_asString() {
        return new String(new java.lang.String(raw, StandardCharsets.UTF_8));
    }

    public void avm_set(ByteArray v) {
        set(v);
    }

    public ByteArray avm_asByteArray() {
        return new ByteArray(raw.clone());
    }

    public void set(byte v) {
        raw = BigInteger.valueOf(v).toByteArray();
    }

    public void set(short v) {
        raw = BigInteger.valueOf(v).toByteArray();
    }

    public void set(int v) {
        raw = BigInteger.valueOf(v).toByteArray();
    }

    public void set(long v) {
        raw = BigInteger.valueOf(v).toByteArray();
    }

    public void set(float v) {
        raw = ByteBuffer.allocate(Float.BYTES).putFloat(v).array();
    }

    public void set(double v) {
        raw = ByteBuffer.allocate(Double.BYTES).putDouble(v).array();
    }

    public void set(char v) {
        raw = BigInteger.valueOf(v).toByteArray();
    }

    public void set(boolean v) {
        raw = BigInteger.valueOf(v ? 1 : 0).toByteArray();
    }

    public void set(s.java.math.BigInteger v) {
        raw = v.getUnderlying().toByteArray();
    }

    public void set(Address v) {
        raw = v.toByteArray();
    }

    public void set(String v) {
        raw = v.getUnderlying().getBytes(StandardCharsets.UTF_8);
    }

    public void set(ByteArray v) {
        raw = v.getUnderlying().clone();
    }

    public byte[] asByteArray() {
        return raw;
    }
}
