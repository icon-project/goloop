package foundation.icon.icx.crypto;

import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import foundation.icon.icx.data.Bytes;

import java.io.File;
import java.io.IOException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.InputMismatchException;
import java.util.TimeZone;


/**
 * Original Code
 * https://github.com/web3j/web3j/blob/master/crypto/src/main/java/org/web3j/crypto/WalletUtils.java
 * Utility functions for working with Keystore files.
 */
public class KeyStoreUtils {

    private KeyStoreUtils() { }

    private static final ObjectMapper objectMapper = new ObjectMapper();

    static {
        objectMapper.configure(JsonParser.Feature.ALLOW_UNQUOTED_FIELD_NAMES, true);
        objectMapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }


    public static String generateWalletFile(
            KeystoreFile file, File destinationDirectory) throws IOException {

        String fileName = getWalletFileName(file);
        File destination = new File(destinationDirectory, fileName);
        objectMapper.writeValue(destination, file);
        return fileName;
    }

    public static Bytes loadPrivateKey(String password, File source)
            throws IOException, KeystoreException {
        ObjectMapper mapper = new ObjectMapper();
        KeystoreFile keystoreFile = mapper.readValue(source, KeystoreFile.class);
        if (keystoreFile.getCoinType() == null || !keystoreFile.getCoinType().equalsIgnoreCase("icx"))
            throw new InputMismatchException("Invalid Keystore file");
        return Keystore.decrypt(password, keystoreFile);
    }

    private static String getWalletFileName(KeystoreFile keystoreFile) {
        SimpleDateFormat dateFormat = new SimpleDateFormat("'UTC--'yyyy-MM-dd'T'HH-mm-ss.SSS'--'");
        dateFormat.setTimeZone(TimeZone.getTimeZone("UTC"));
        return dateFormat.format(new Date()) + keystoreFile.getAddress() + ".json";
    }

}
