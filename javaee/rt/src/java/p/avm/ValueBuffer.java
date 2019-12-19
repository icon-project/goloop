package p.avm;

import a.ByteArray;
import i.IInstrumentation;
import org.aion.avm.RuntimeMethodFeeSchedule;
import s.java.lang.Object;
import s.java.lang.String;

import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;

public class ValueBuffer extends Object implements Value {
    private byte[] raw;

    public ValueBuffer() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        raw = new byte[0];
    }

    public ValueBuffer(byte v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(short v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(int v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(long v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(float v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(double v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(char v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(boolean v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(s.java.math.BigInteger v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(Address v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(String v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    public ValueBuffer(ByteArray v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_constructor);
        set(v);
    }

    private ValueBuffer(byte[] v) {
        set(v);
    }

    public void set(byte[] raw) {
        this.raw = raw;
    }

    public ValueBuffer avm_set(byte v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public byte avm_asByte() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return (new BigInteger(raw)).byteValue();
    }

    public ValueBuffer avm_set(short v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public short avm_asShort() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return (new BigInteger(raw)).shortValue();
    }

    public ValueBuffer avm_set(int v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public int avm_asInt() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return asInt();
    }

    public int asInt() {
        return (new BigInteger(raw)).intValue();
    }

    public ValueBuffer avm_set(long v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public long avm_asLong() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return (new BigInteger(raw)).longValue();
    }

    public ValueBuffer avm_set(float v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public float avm_asFloat() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return ByteBuffer.wrap(raw).getFloat();
    }

    public ValueBuffer avm_set(double v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public double avm_asDouble() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return ByteBuffer.wrap(raw).getDouble();
    }

    public ValueBuffer avm_set(char v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public char avm_asChar() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return (char) (new BigInteger(raw)).intValue();
    }

    public ValueBuffer avm_set(boolean v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public boolean avm_asBoolean() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return (new BigInteger(raw)).intValue() != 0;
    }

    public ValueBuffer avm_set(s.java.math.BigInteger v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public s.java.math.BigInteger avm_asBigInteger() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return new s.java.math.BigInteger(new BigInteger(raw));
    }

    public ValueBuffer avm_set(Address v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public Address avm_asAddress() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return new Address(raw);
    }

    public ValueBuffer avm_set(String v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public String avm_asString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
        return new String(new java.lang.String(raw, StandardCharsets.UTF_8));
    }

    public ValueBuffer avm_set(ByteArray v) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_set);
        set(v);
        return this;
    }

    public ByteArray avm_asByteArray() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.ValueBuffer_avm_get);
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
