package org.aion.data;

import java.io.File;
import java.io.IOException;
import java.math.BigInteger;
import java.nio.file.Files;
import java.nio.file.NoSuchFileException;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;

import org.aion.avm.core.util.ByteArrayWrapper;
import org.aion.avm.core.util.Helpers;
import i.RuntimeAssertionError;

public class DirectoryBackedAccountStore implements IAccountStore {
    private static final String FILE_NAME_CODE = "code";
    private static final String FILE_NAME_TRANSFORMED_CODE = "transformed_code";

    private static final String FILE_NAME_BALANCE = "balance";
    private static final String FILE_NAME_NONCE = "nonce";
    private static final String FILE_PREFIX_KEY = "key_";
    private static final String FILE_GRAPH = "graph";

    private final File accountDirectory;
    public DirectoryBackedAccountStore(File accountDirectory) {
        this.accountDirectory = accountDirectory;
    }

    @Override
    public byte[] getCode() {
        return readFile(FILE_NAME_CODE);
    }

    @Override
    public void setCode(byte[] code) {
        writeFile(FILE_NAME_CODE, code);
    }

    @Override
    public byte[] getTransformedCode() {
        return readFile(FILE_NAME_TRANSFORMED_CODE);
    }

    @Override
    public void setTransformedCode(byte[] code) {
        writeFile(FILE_NAME_TRANSFORMED_CODE, code);
    }

    @Override
    public BigInteger getBalance() {
        byte[] data = readFile(FILE_NAME_BALANCE);
        // In the future, we probably want to force these to be written before being read and avoid this null check.
        return (null != data)
                ? new BigInteger(data)
                : BigInteger.ZERO;
    }

    @Override
    public void setBalance(BigInteger balance) {
        byte[] data = balance.toByteArray();
        writeFile(FILE_NAME_BALANCE, data);
    }

    @Override
    public long getNonce() {
        byte[] data = readFile(FILE_NAME_NONCE);
        // In the future, we probably want to force these to be written before being read and avoid this null check.
        return (null != data)
                ? decodeLong(data)
                : 0L;
    }

    @Override
    public void setNonce(long nonce) {
        byte[] data = encodeLong(nonce);
        writeFile(FILE_NAME_NONCE, data);
    }

    @Override
    public byte[] getData(byte[] key) {
        return readFile(fileNameForKey(key));
    }

    @Override
    public void setData(byte[] key, byte[] value) {
        writeFile(fileNameForKey(key), value);
    }

    @Override
    public void removeData(byte[] key) {
        deleteFile(fileNameForKey(key));
    }

    @Override
    public Map<ByteArrayWrapper, byte[]> getStorageEntries() {
        Map<ByteArrayWrapper, byte[]> result = new HashMap<>();
        // List the files and parse any names with FILE_PREFIX_KEY as a prefix.
        for (File file : this.accountDirectory.listFiles()) {
            String name = file.getName();
            if (name.startsWith(FILE_PREFIX_KEY)) {
                String hexOfKey = name.substring(FILE_PREFIX_KEY.length());
                // 2 chars per encoded byte so this MUST be an even length.
                RuntimeAssertionError.assertTrue(0 == (hexOfKey.length() % 2));
                byte[] key = new byte[hexOfKey.length() / 2];
                for (int i = 0; i < hexOfKey.length(); i += 2) {
                    String bite = hexOfKey.substring(i, i+2);
                    // Parse into an int, since we need these to be effectively unsigned.
                    int largeByte = Integer.parseInt(bite, 16);
                    key[i/2] = (byte)(0xff & largeByte);
                }
                byte[] value = readFile(name);
                result.put(new ByteArrayWrapper(key), value);
            }
        }
        return result;
    }

    @Override
    public void setObjectGraph(byte[] data) {
        writeFile(FILE_GRAPH, data);
    }

    @Override
    public byte[] getObjectGraph() {
        return readFile(FILE_GRAPH);
    }


    private byte[] readFile(String fileName) {
        Path oneFile = new File(this.accountDirectory, fileName).toPath();
        try {
            return Files.readAllBytes(oneFile);
        } catch (NoSuchFileException e) {
            // In the future, we probably want to force these to be written before being read.
            return null;
        } catch (IOException e) {
            // This implementation doesn't handle exceptions.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    private void writeFile(String fileName, byte[] data) {
        Path oneFile = new File(this.accountDirectory, fileName).toPath();
        try {
            Files.write(oneFile, data);
        } catch (IOException e) {
            // This implementation doesn't handle exceptions.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    private void deleteFile(String fileName) {
        Path oneFile = new File(this.accountDirectory, fileName).toPath();
        try {
            Files.deleteIfExists(oneFile);
        } catch (IOException e) {
            // This implementation doesn't handle exceptions.
            throw RuntimeAssertionError.unexpected(e);
        }
    }

    private long decodeLong(byte[] data) {
        long value = (
                ((long)(0xff & data[0]) << 56)
              | ((long)(0xff & data[1]) << 48)
              | ((long)(0xff & data[2]) << 40)
              | ((long)(0xff & data[3]) << 32)
              | ((long)(0xff & data[4]) << 24)
              | ((long)(0xff & data[5]) << 16)
              | ((long)(0xff & data[6]) << 8)
              | ((long)(0xff & data[7]) << 0)
        );
        return value;
    }

    private byte[] encodeLong(long value) {
        byte[] encoded = new byte[Long.BYTES];
        encoded[0] = (byte) (0xff & (value >> 56));
        encoded[1] = (byte) (0xff & (value >> 48));
        encoded[2] = (byte) (0xff & (value >> 40));
        encoded[3] = (byte) (0xff & (value >> 32));
        encoded[4] = (byte) (0xff & (value >> 24));
        encoded[5] = (byte) (0xff & (value >> 16));
        encoded[6] = (byte) (0xff & (value >>  8));
        encoded[7] = (byte) (0xff & (value >>  0));
        return encoded;
    }

    private String fileNameForKey(byte[] key) {
        // We need to make sure that this key isn't going to hit some limit since we are back-ending on the filesystem.
        // This is an AssertionError since it is a limitation of this testing implementation, not a usage error.
        RuntimeAssertionError.assertTrue(key.length <= 32);
        return FILE_PREFIX_KEY + Helpers.bytesToHexString(key);
    }
}
