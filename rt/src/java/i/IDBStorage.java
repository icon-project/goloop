package i;

import p.avm.Value;
import p.avm.ValueBuffer;

public interface IDBStorage {
    void setTyped(byte[] key, IObject value);
    IObject getTyped(byte[] key);
    void setValue(byte[] key, Value value);
    Value getValue(byte[] key, ValueBuffer out);
    void setArrayLength(byte[] key, int l);
    int getArrayLength(byte[] key);
    void flush();
}
