package i;

import p.avm.ValueBuffer;

public interface IDBStorage {
    void setValue(byte[] key, IObject value);
    IObject getValue(byte[] key);
    ValueBuffer getValue(byte[] key, ValueBuffer out);
    void setArrayLength(byte[] hashedKey, int l);
    int getArrayLength(byte[] hashedKey);
    void flush();
}
