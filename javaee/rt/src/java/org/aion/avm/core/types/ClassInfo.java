package org.aion.avm.core.types;

public class ClassInfo {
    private boolean isInterface;
    private byte[] bytes;

    public ClassInfo(boolean isInterface, byte[] bytes) {
        this.isInterface = isInterface;
        this.bytes = bytes;
    }

    public boolean isInterface() {
        return isInterface;
    }

    public byte[] getBytes() {
        return bytes;
    }
}
