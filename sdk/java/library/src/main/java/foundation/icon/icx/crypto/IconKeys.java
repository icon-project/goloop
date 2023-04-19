package foundation.icon.icx.crypto;

import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import org.bouncycastle.crypto.RuntimeCryptoException;
import org.bouncycastle.jcajce.provider.asymmetric.ec.BCECPrivateKey;
import org.bouncycastle.jcajce.provider.digest.SHA3;
import org.bouncycastle.jce.ECNamedCurveTable;
import org.bouncycastle.jce.provider.BouncyCastleProvider;
import org.bouncycastle.jce.spec.ECNamedCurveParameterSpec;
import org.bouncycastle.math.ec.ECPoint;
import org.bouncycastle.util.BigIntegers;
import org.bouncycastle.util.encoders.Hex;

import java.math.BigInteger;
import java.security.InvalidAlgorithmParameterException;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.NoSuchProviderException;
import java.security.Provider;
import java.security.SecureRandom;
import java.security.Security;
import java.security.spec.ECGenParameterSpec;
import java.util.Arrays;

/**
 * Implementation from
 * https://github.com/web3j/web3j/blob/master/crypto/src/main/java/org/web3j/crypto/Keys.java
 * Crypto key utilities.
 */
@SuppressWarnings({"WeakerAccess", "unused"})
public class IconKeys {

    public static final int PRIVATE_KEY_SIZE = 32;
    public static final int PUBLIC_KEY_SIZE = 65;
    public static final int PUBLIC_KEY_SIZE_COMP = 33;

    public static final int ADDRESS_SIZE = 160;
    public static final int ADDRESS_LENGTH_IN_HEX = ADDRESS_SIZE >> 2;
    private static final SecureRandom SECURE_RANDOM;
    private static int isAndroid = -1;

    public static final double MIN_BOUNCY_CASTLE_VERSION = 1.46;

    static {
        if (isAndroidRuntime()) {
            new LinuxSecureRandom();
        }

        Provider provider = Security.getProvider(BouncyCastleProvider.PROVIDER_NAME);
        Provider newProvider = new BouncyCastleProvider();

        if (newProvider.getVersion() < MIN_BOUNCY_CASTLE_VERSION) {
            String message = String.format(
                    "The version of BouncyCastle should be %f or newer", MIN_BOUNCY_CASTLE_VERSION);
            throw new RuntimeCryptoException(message);
        }

        if (provider != null) {
            Security.removeProvider(BouncyCastleProvider.PROVIDER_NAME);
        }

        Security.addProvider(newProvider);

        SECURE_RANDOM = new SecureRandom();
    }

    private IconKeys() {
    }

    public static Bytes createPrivateKey() throws InvalidAlgorithmParameterException, NoSuchAlgorithmException, NoSuchProviderException {
        KeyPairGenerator keyPairGenerator = KeyPairGenerator.getInstance("EC", "BC");
        ECGenParameterSpec ecGenParameterSpec = new ECGenParameterSpec("secp256k1");
        keyPairGenerator.initialize(ecGenParameterSpec, secureRandom());
        KeyPair keyPair = keyPairGenerator.generateKeyPair();
        BigInteger d = ((BCECPrivateKey) keyPair.getPrivate()).getD();
        return new Bytes(BigIntegers.asUnsignedByteArray(PRIVATE_KEY_SIZE, d));
    }

    public static Bytes privateKeyToPublicKey(Bytes privateKey, boolean compressed) {
        ECNamedCurveParameterSpec spec = ECNamedCurveTable.getParameterSpec("secp256k1");
        ECPoint pointQ = spec.getG().multiply(new BigInteger(1, privateKey.toByteArray()));
        return new Bytes(pointQ.getEncoded(compressed));
    }

    public static Bytes getPublicKey(Bytes privateKey) {
        return privateKeyToPublicKey(privateKey, false);
    }

    public static Bytes getPublicKeyCompressed(Bytes privateKey) {
        return privateKeyToPublicKey(privateKey, true);
    }

    public static Bytes convertPublicKey(Bytes pubKey, boolean toCompressed) {
        int inputLen = pubKey.length();
        int expectInputLen = toCompressed ? PUBLIC_KEY_SIZE : PUBLIC_KEY_SIZE_COMP;
        int resultLen = toCompressed ? PUBLIC_KEY_SIZE_COMP : PUBLIC_KEY_SIZE;

        if (inputLen == resultLen) {
            return pubKey;
        }
        if (inputLen != expectInputLen) {
            throw new IllegalArgumentException("The length of Bytes must be " + PUBLIC_KEY_SIZE + " or " + PUBLIC_KEY_SIZE_COMP);
        }
        ECNamedCurveParameterSpec spec = ECNamedCurveTable.getParameterSpec("secp256k1");
        ECPoint point = spec.getCurve().decodePoint(pubKey.toByteArray());
        return new Bytes(point.getEncoded(toCompressed));
    }

    public static Bytes publicKeyToUncompressed(Bytes pubKey) {
        return convertPublicKey(pubKey, false);
    }

    public static Bytes publicKeyToCompressed(Bytes pubKey) {
        return convertPublicKey(pubKey, true);
    }

    public static Address getAddress(Bytes publicKey) {
        return new Address(Address.AddressPrefix.EOA, getAddressHash(publicKey.toByteArray()));
    }

    public static byte[] getAddressHash(BigInteger publicKey) {
        if (publicKey.signum() < 0) {
            throw new IllegalArgumentException("The publicKey cannot be negative");
        }
        return getAddressHash(BigIntegers.asUnsignedByteArray(PUBLIC_KEY_SIZE, publicKey));
    }

    public static byte[] getAddressHash(byte[] publicKey) {
        byte[] pubKey = publicKeyToUncompressed(new Bytes(publicKey)).toByteArray();
        // remove a constant prefix (0x04)
        // https://github.com/bcgit/bc-java/blob/master/core/src/main/java/org/bouncycastle/math/ec/ECPoint.java#L489
        byte[] pub = Arrays.copyOfRange(pubKey, 1, pubKey.length);
        byte[] hash = new SHA3.Digest256().digest(pub);

        int length = 20;
        byte[] result = new byte[20];
        System.arraycopy(hash, hash.length - 20, result, 0, length);
        return result;
    }

    public static boolean isValidAddress(Address input) {
        return isValidAddress(input.toString());
    }

    public static boolean isValidAddress(String input) {
        String cleanInput = cleanHexPrefix(input);
        try {
            return cleanInput.matches("^[0-9a-f]{40}$") && cleanInput.length() == ADDRESS_LENGTH_IN_HEX;
        } catch (NumberFormatException e) {
            return false;
        }
    }

    public static boolean isValidAddressBody(byte[] body) {
        return body.length == 20 &&
                IconKeys.isValidAddress(Hex.toHexString(body));
    }

    public static boolean isContractAddress(Address address) {
        return address.getPrefix() == Address.AddressPrefix.CONTRACT;
    }

    public static String cleanHexPrefix(String input) {
        if (containsHexPrefix(input)) {
            return input.substring(2);
        } else {
            return input;
        }
    }

    public static boolean containsHexPrefix(String input) {
        return getAddressHexPrefix(input) != null;
    }

    public static Address.AddressPrefix getAddressHexPrefix(String input) {
        return Address.AddressPrefix.fromString(input.substring(0, 2));
    }

    public static SecureRandom secureRandom() {
        return SECURE_RANDOM;
    }

    public static boolean isAndroidRuntime() {
        if (isAndroid == -1) {
            final String runtime = System.getProperty("java.runtime.name");
            isAndroid = (runtime != null && runtime.equals("Android Runtime")) ? 1 : 0;
        }
        return isAndroid == 1;
    }
}
