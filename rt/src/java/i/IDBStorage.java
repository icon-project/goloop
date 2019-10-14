package i;

import p.avm.PrimitiveBuffer;

public interface IDBStorage {
    void setValue(byte[] key, IObject value);
    IObject getValue(byte[] key);
    PrimitiveBuffer getValue(byte[] key, PrimitiveBuffer out);
    void setArrayLength(byte[] hashedKey, int l);
    int getArrayLength(byte[] hashedKey);
    void flush();
}
