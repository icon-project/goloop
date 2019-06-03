package foundation.icon.icx.crypto;

/**
 * Original Code
 * https://github.com/web3j/web3j/blob/master/crypto/src/main/java/org/web3j/crypto/CipherException.java
 */
public class KeystoreException extends Exception {

    KeystoreException(String message) {
        super(message);
    }

    KeystoreException(Throwable cause) {
        super(cause);
    }

    KeystoreException(String message, Throwable cause) {
        super(message, cause);
    }
}
