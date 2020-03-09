package i;

public interface IDBStorage {
    void setBytes(byte[] key, byte[] value);
    byte[] getBytes(byte[] key);
    void setArrayLength(byte[] key, int l);
    int getArrayLength(byte[] key);
    void flush();
}
