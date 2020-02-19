package i;

public interface IDBStorage {
    void setTyped(byte[] key, IObject value);
    IObject getTyped(byte[] key);
    void setBytes(byte[] key, byte[] value);
    byte[] getBytes(byte[] key);
    void setArrayLength(byte[] key, int l);
    int getArrayLength(byte[] key);
    void flush();
}
