package avm;

import java.math.BigInteger;

public interface Value {
    byte asByte();
    short asShort();
    int asInt();
    long asLong();
    float asFloat();
    double asDouble();
    char asChar();
    boolean asBoolean();
    BigInteger asBigInteger();
    Address asAddress();
    String asString();
    byte[] asByteArray();
}
