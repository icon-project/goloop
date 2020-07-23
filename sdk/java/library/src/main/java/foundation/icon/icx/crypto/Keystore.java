package foundation.icon.icx.crypto;

import foundation.icon.icx.data.Bytes;
import org.bouncycastle.crypto.digests.SHA256Digest;
import org.bouncycastle.crypto.generators.PKCS5S2ParametersGenerator;
import org.bouncycastle.crypto.generators.SCrypt;
import org.bouncycastle.crypto.params.KeyParameter;
import org.bouncycastle.jcajce.provider.digest.Keccak;
import org.bouncycastle.util.encoders.Hex;

import javax.crypto.BadPaddingException;
import javax.crypto.Cipher;
import javax.crypto.IllegalBlockSizeException;
import javax.crypto.NoSuchPaddingException;
import javax.crypto.spec.IvParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.InvalidAlgorithmParameterException;
import java.security.InvalidKeyException;
import java.security.NoSuchAlgorithmException;
import java.util.Arrays;
import java.util.UUID;

import static foundation.icon.icx.crypto.IconKeys.secureRandom;

/**
 * Original Code
 * https://github.com/web3j/web3j/blob/master/crypto/src/main/java/org/web3j/crypto/Wallet.java
 *
 * <p>Ethereum wallet file management. For reference, refer to
 * <a href="https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition">
 * Web3 Secret Storage Definition</a> or the
 * <a href="https://github.com/ethereum/go-ethereum/blob/master/accounts/key_store_passphrase.go">
 * Go Ethereum client implementation</a>.</p>
 *
 * <p><strong>Note:</strong> the Bouncy Castle Scrypt implementation
 * {@link SCrypt}, fails to comply with the following
 * Ethereum reference
 * <a href="https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition#scrypt">
 * Scrypt test vector</a>:</p>
 *
 * <pre>
 * {@code
 * // Only value of r that cost (as an int) could be exceeded for is 1
 * if (r == 1 && N_STANDARD > 65536)
 * {
 *     throw new IllegalArgumentException("Cost parameter N_STANDARD must be > 1 and < 65536.");
 * }
 * }
 * </pre>
 */
public class Keystore {

    private static final int N_LIGHT = 1 << 12;
    private static final int P_LIGHT = 6;

    private static final int N_STANDARD = 1 << 18;
    private static final int P_STANDARD = 1;

    private static final int R = 8;
    private static final int DKLEN = 32;

    private static final int CURRENT_VERSION = 3;

    private static final String CIPHER = "aes-128-ctr";
    static final String AES_128_CTR = "pbkdf2";
    static final String SCRYPT = "scrypt";

    public static KeystoreFile create(String password, Bytes privateKey, int n, int p)
            throws KeystoreException {

        byte[] salt = generateRandomBytes(32);

        byte[] derivedKey = generateDerivedScryptKey(
                password.getBytes(StandardCharsets.UTF_8), salt, n, R, p, DKLEN);

        byte[] encryptKey = Arrays.copyOfRange(derivedKey, 0, 16);
        byte[] iv = generateRandomBytes(16);

        byte[] privateKeyBytes = privateKey.toByteArray(IconKeys.PRIVATE_KEY_SIZE);

        byte[] cipherText = performCipherOperation(
                Cipher.ENCRYPT_MODE, iv, encryptKey, privateKeyBytes);

        byte[] mac = generateMac(derivedKey, cipherText);

        return createWalletFile(privateKey, cipherText, iv, salt, mac, n, p);
    }

    private static KeystoreFile createWalletFile(
            Bytes privateKey, byte[] cipherText, byte[] iv, byte[] salt, byte[] mac,
            int n, int p) {

        KeystoreFile keystoreFile = new KeystoreFile();
        keystoreFile.setAddress(IconKeys.getAddress(IconKeys.getPublicKey(privateKey)));

        KeystoreFile.Crypto crypto = new KeystoreFile.Crypto();
        crypto.setCipher(CIPHER);
        crypto.setCiphertext(Hex.toHexString(cipherText));
        keystoreFile.setCrypto(crypto);

        KeystoreFile.CipherParams cipherParams = new KeystoreFile.CipherParams();
        cipherParams.setIv(Hex.toHexString(iv));
        crypto.setCipherparams(cipherParams);

        crypto.setKdf(SCRYPT);
        KeystoreFile.ScryptKdfParams kdfParams = new KeystoreFile.ScryptKdfParams();
        kdfParams.setDklen(DKLEN);
        kdfParams.setN(n);
        kdfParams.setP(p);
        kdfParams.setR(R);
        kdfParams.setSalt(Hex.toHexString(salt));
        crypto.setKdfparams(kdfParams);

        crypto.setMac(Hex.toHexString(mac));
        keystoreFile.setCrypto(crypto);
        keystoreFile.setId(UUID.randomUUID().toString());
        keystoreFile.setVersion(CURRENT_VERSION);
        keystoreFile.setCoinType("icx");
        return keystoreFile;
    }

    private static byte[] generateDerivedScryptKey(
            byte[] password, byte[] salt, int n, int r, int p, int dkLen) {
        return SCrypt.generate(password, salt, n, r, p, dkLen);
    }

    private static byte[] generateAes128CtrDerivedKey(
            byte[] password, byte[] salt, int c, String prf) throws KeystoreException {

        if (!prf.equals("hmac-sha256")) {
            throw new KeystoreException("Unsupported prf:" + prf);
        }

        // Java 8 supports this, but you have to convert the password to a character array, see
        // http://stackoverflow.com/a/27928435/3211687

        PKCS5S2ParametersGenerator gen = new PKCS5S2ParametersGenerator(new SHA256Digest());
        gen.init(password, salt, c);
        return ((KeyParameter) gen.generateDerivedParameters(256)).getKey();
    }

    private static byte[] performCipherOperation(
            int mode, byte[] iv, byte[] encryptKey, byte[] text) throws KeystoreException {

        try {
            IvParameterSpec ivParameterSpec = new IvParameterSpec(iv);
            Cipher cipher = Cipher.getInstance("AES/CTR/NoPadding");

            SecretKeySpec secretKeySpec = new SecretKeySpec(encryptKey, "AES");
            cipher.init(mode, secretKeySpec, ivParameterSpec);
            return cipher.doFinal(text);
        } catch (NoSuchPaddingException | NoSuchAlgorithmException
                | InvalidAlgorithmParameterException | InvalidKeyException
                | BadPaddingException | IllegalBlockSizeException e) {
            throw new KeystoreException("Error performing cipher operation", e);
        }
    }

    private static byte[] generateMac(byte[] derivedKey, byte[] cipherText) {
        byte[] result = new byte[16 + cipherText.length];

        System.arraycopy(derivedKey, 16, result, 0, 16);
        System.arraycopy(cipherText, 0, result, 16, cipherText.length);

        Keccak.DigestKeccak kecc = new Keccak.Digest256();
        kecc.update(result, 0, result.length);
        return kecc.digest();
    }

    public static Bytes decrypt(String password, KeystoreFile keystoreFile)
            throws KeystoreException {

        validate(keystoreFile);

        KeystoreFile.Crypto crypto = keystoreFile.getCrypto();

        byte[] mac = Hex.decode(crypto.getMac());
        byte[] iv = Hex.decode(crypto.getCipherparams().getIv());
        byte[] cipherText = Hex.decode(crypto.getCiphertext());

        byte[] derivedKey;

        KeystoreFile.KdfParams kdfParams = crypto.getKdfparams();
        if (kdfParams instanceof KeystoreFile.ScryptKdfParams) {
            KeystoreFile.ScryptKdfParams scryptKdfParams =
                    (KeystoreFile.ScryptKdfParams) crypto.getKdfparams();
            int dklen = scryptKdfParams.getDklen();
            int n = scryptKdfParams.getN();
            int p = scryptKdfParams.getP();
            int r = scryptKdfParams.getR();
            byte[] salt = Hex.decode(scryptKdfParams.getSalt());
            derivedKey = generateDerivedScryptKey(password.getBytes(StandardCharsets.UTF_8), salt, n, r, p, dklen);
        } else if (kdfParams instanceof KeystoreFile.Aes128CtrKdfParams) {
            KeystoreFile.Aes128CtrKdfParams aes128CtrKdfParams =
                    (KeystoreFile.Aes128CtrKdfParams) crypto.getKdfparams();
            int c = aes128CtrKdfParams.getC();
            String prf = aes128CtrKdfParams.getPrf();
            byte[] salt = Hex.decode(aes128CtrKdfParams.getSalt());

            derivedKey = generateAes128CtrDerivedKey(password.getBytes(StandardCharsets.UTF_8), salt, c, prf);
        } else {
            throw new KeystoreException("Unable to deserialize params: " + crypto.getKdf());
        }

        byte[] derivedMac = generateMac(derivedKey, cipherText);

        if (!Arrays.equals(derivedMac, mac)) {
            throw new KeystoreException("Invalid password provided");
        }

        byte[] encryptKey = Arrays.copyOfRange(derivedKey, 0, 16);
        byte[] privateKey = performCipherOperation(Cipher.DECRYPT_MODE, iv, encryptKey, cipherText);
        return new Bytes(privateKey);
    }

    private static void validate(KeystoreFile keystoreFile) throws KeystoreException {
        KeystoreFile.Crypto crypto = keystoreFile.getCrypto();

        if (keystoreFile.getVersion() != CURRENT_VERSION) {
            throw new KeystoreException("Keystore version is not supported");
        }

        if (!crypto.getCipher().equals(CIPHER)) {
            throw new KeystoreException("Keystore cipher is not supported");
        }

        if (!crypto.getKdf().equals(AES_128_CTR) && !crypto.getKdf().equals(SCRYPT)) {
            throw new KeystoreException("KDF type is not supported");
        }

        if (keystoreFile.getCoinType() == null || !keystoreFile.getCoinType().equalsIgnoreCase("icx"))
            throw new KeystoreException("Invalid Keystore file");
    }

    private static byte[] generateRandomBytes(int size) {
        byte[] bytes = new byte[size];
        secureRandom().nextBytes(bytes);
        return bytes;
    }
}
