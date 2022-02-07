/*
 * Copyright 2019 ICON Foundation
 * Copyright (c) 2018 Aion Foundation https://aion.network/
 */

package p.score;

import a.ByteArray;
import i.IInstrumentation;
import i.IObject;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import org.aion.avm.RuntimeMethodFeeSchedule;

/**
 * The address has a very specific meaning, within the environment, so we wrap a ByteArray to produce this more specific type.
 *
 * This is likely to change a lot as we build more DApp tests (see issue-76 for more details on how we might want to evolve this).
 * There is a good chance that we will convert this into an interface so that our implementation can provide a richer interface to
 * our AVM code than we want to support for the contract.
 */
public final class Address extends s.java.lang.Object {
    // Runtime-facing implementation.
    public static final int avm_LENGTH = 21;

    // Note that we always contain an internal byte[] and we serialize that, specially.
    private final byte[] internalArray = new byte[avm_LENGTH];

    /**
     * The constructor which user code can call, directly, to create an Address object.
     * This will remain until/unless we decide to make a factory which creates these from within the runtime.
     *
     * @param raw The raw bytes representing the address.
     */
    public Address(ByteArray raw) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_constructor);
        if (null == raw) {
            throw new NullPointerException();
        }
        setUnderlying(raw.getUnderlying());
    }

    public static Address avm_fromString(s.java.lang.String str) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_fromString);
        if (str == null) {
            throw new NullPointerException();
        }
        String raw = str.getUnderlying();
        if (raw.length() != avm_LENGTH * 2) {
            throw new IllegalArgumentException();
        }
        if (raw.startsWith("hx") || raw.startsWith("cx")) {
            byte[] bytes = new byte[avm_LENGTH];
            bytes[0] = (byte) (raw.startsWith("hx") ? 0x0 : 0x1);
            for (int i = 1; i < avm_LENGTH; i++) {
                int j = i * 2;
                bytes[i] = (byte) Integer.parseInt(raw.substring(j, j + 2), 16);
            }
            return newWithCharge(bytes);
        } else {
            throw new IllegalArgumentException();
        }
    }

    public boolean avm_isContract() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_unwrap);
        lazyLoad();
        return internalArray[0] == 0x1;
    }

    /**
     * Similarly, this method will probably be removed or otherwise hidden.
     *
     * @return The raw bytes underneath the address.
     */
    public ByteArray avm_toByteArray() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_unwrap);
        lazyLoad();
        byte[] copy = copyOfInternal();
        return new ByteArray(copy);
    }

    @Override
    public int avm_hashCode() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_hashCode);

        return internalHashCode();
    }

    @Override
    public boolean avm_equals(IObject obj) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_equals);

        return internalEquals(obj);
    }

    @Override
    public s.java.lang.String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Address_avm_toString);
        lazyLoad();
        char[] hexChars = toHexChars(this.internalArray);
        return new s.java.lang.String(new java.lang.String(hexChars));
    }

    @Override
    public String toString() {
        lazyLoad();
        char[] hexChars = toHexChars(this.internalArray);
        return new String(hexChars);
    }

    private static char[] toHexChars(byte[] bytes) {
        char[] hexChars = new char[bytes.length * 2];
        if (bytes[0] == 0x0) {
            hexChars[0] = 'h';
        } else {
            hexChars[0] = 'c';
        }
        hexChars[1] = 'x';
        for (int i = 1; i < bytes.length; i++) {
            int v = bytes[i] & 0xFF;
            hexChars[i * 2] = hexArray[v >>> 4];
            hexChars[i * 2 + 1] = hexArray[v & 0x0F];
        }
        return hexChars;
    }

    private static final char[] hexArray = "0123456789abcdef".toCharArray();

    // Compiler-facing implementation.
    public static final int LENGTH = avm_LENGTH;

    /**
     * Note that this constructor is only here to support our tests while we decide whether or not to expose the constructor
     * of construct the class this way.
     *
     * @param raw The raw bytes representing the address.
     */
    public Address(byte[] raw) {
        if (null == raw) {
            throw new NullPointerException();
        }
        setUnderlying(raw);
    }

    public static Address newWithCharge(byte[] raw) {
        IInstrumentation.charge(RuntimeMethodFeeSchedule.Address_avm_constructor);
        return new Address(raw);
    }

    /**
     * Similarly, this method will probably be removed or otherwise hidden.
     *
     * @return The raw bytes underneath the address.
     */
    public byte[] toByteArray() {
        lazyLoad();
        return copyOfInternal();
    }

    @Override
    public boolean equals(java.lang.Object obj) {
        return internalEquals(obj);
    }

    @Override
    public int hashCode() {
        return internalHashCode();
    }

    // Support for deserialization
    public Address(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(Address.class, deserializer);
        for (int i = 0; i < avm_LENGTH; ++i) {
            this.internalArray[i] = deserializer.readByte();
        }
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(Address.class, serializer);
        for (int i = 0; i < avm_LENGTH; ++i) {
            serializer.writeByte(this.internalArray[i]);
        }
    }

    private void setUnderlying(byte[] raw) {
        int offset = 0;
        if (raw.length == avm_LENGTH - 1) {
            offset = 1;
        } else if (raw.length != avm_LENGTH) {
            throw new IllegalArgumentException();
        }
        System.arraycopy(raw, 0, this.internalArray, offset, raw.length);
    }

    private byte[] copyOfInternal() {
        byte[] copy = new byte[avm_LENGTH];
        System.arraycopy(this.internalArray, 0, copy, 0, avm_LENGTH);
        return copy;
    }

    private int internalHashCode() {
        int code = 0;
        lazyLoad();
        for (byte elt : this.internalArray) {
            code += (int)elt;
        }
        return code;
    }

    private boolean internalEquals(java.lang.Object obj) {
        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof Address)) {
            Address other = (Address) obj;
            lazyLoad();
            other.lazyLoad();
            isEqual = true;
            for (int i = 0; isEqual && (i < avm_LENGTH); ++i) {
                isEqual = (this.internalArray[i] == other.internalArray[i]);
            }
        }
        return isEqual;
    }
}
