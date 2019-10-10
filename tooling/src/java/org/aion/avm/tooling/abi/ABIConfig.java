package org.aion.avm.tooling.abi;

public class ABIConfig {

    private static ABIConfig instance = null;
    public static final int LATEST_VERSION = 1;

    private ABIConfig() {}

    public static ABIConfig getInstance() {
        if (instance == null) {
            instance = new ABIConfig();
        }
        return instance;
    }

    public boolean isBigIntegerEnabled(int compiledVersion) {
        return compiledVersion >= 1;
    }
}