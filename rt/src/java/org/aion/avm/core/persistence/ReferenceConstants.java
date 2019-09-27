package org.aion.avm.core.persistence;


/**
 * Constants used when describing inter-object references within the serialized form.
 */
public class ReferenceConstants {
    public static final byte REF_NULL = 0x0;
    public static final byte REF_CLASS = 0x1;
    public static final byte REF_CONSTANT = 0x2;
    public static final byte REF_NORMAL = 0x3;
}
