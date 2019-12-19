package foundation.icon.ee.utils;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

public class Crypto {
    public static byte[] sha3_256(byte[] msg) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA3-256");
            return digest.digest(msg);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }
}
