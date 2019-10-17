package p.avm;

import s.java.math.BigInteger;
import s.java.lang.String;
import a.ByteArray;

public interface Value {
    byte avm_asByte();
    short avm_asShort();
    int avm_asInt();
    long avm_asLong();
    float avm_asFloat();
    double avm_asDouble();
    char avm_asChar();
    boolean avm_asBoolean();
    BigInteger avm_asBigInteger();
    Address avm_asAddress();
    String avm_asString();
    ByteArray avm_asByteArray();
}
