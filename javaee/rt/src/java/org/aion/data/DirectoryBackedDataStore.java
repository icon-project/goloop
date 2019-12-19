package org.aion.data;

import java.io.File;

import org.aion.avm.core.util.Helpers;
import i.RuntimeAssertionError;


public class DirectoryBackedDataStore implements IDataStore {
    private final File topLevelDirectory;
    public DirectoryBackedDataStore(File topLevelDirectory) {
        this.topLevelDirectory = topLevelDirectory;
    }

    @Override
    public IAccountStore openAccount(byte[] address) {
        File directory = getSubDirectory(address);
        return directory.isDirectory()
                ? new DirectoryBackedAccountStore(directory)
                : null;
    }

    @Override
    public IAccountStore createAccount(byte[] address) {
        File directory = getSubDirectory(address);
        return directory.mkdir()
                ? new DirectoryBackedAccountStore(directory)
                : null;
    }

    @Override
    public void deleteAccount(byte[] address) {
        File directory = getSubDirectory(address);
        for (File file : directory.listFiles()) {
            // The account structure is flat so this better be just a regular file.
            RuntimeAssertionError.assertTrue(file.isFile());
            boolean didDelete = file.delete();
            RuntimeAssertionError.assertTrue(didDelete);
        }
        boolean didDelete = directory.delete();
        RuntimeAssertionError.assertTrue(didDelete);
    }


    private File getSubDirectory(byte[] address) {
        // We need to make sure that this address isn't going to hit some limit (we can tighten this to the specific address length but not all
        // our tests use that).
        if ((null == address) || (address.length < 4) || (address.length > 32)) {
            throw new IllegalArgumentException("Address length incorrect (must be between 4 and 32)");
        }
        String directoryName = "account_" + Helpers.bytesToHexString(address);
        return new File(this.topLevelDirectory, directoryName);
    }
}
