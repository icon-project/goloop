package org.aion.parallel;

/**
 * A package private class wraps address byte array.
 *
 * This class is to remove the requirement of attaching dumpy helper to use {@link p.avm.Address}.
 */
class AddressWrapper {

    private byte[] addr;

    AddressWrapper(byte[] addr){
        this.addr = addr;
    }

    @Override
    public int hashCode() {
        int code = 0;
        for (byte elt : this.addr) {
            code += (int)elt;
        }
        return code;
    }

    @Override
    public boolean equals(Object obj) {
        boolean isEqual = this == obj;
        if (!isEqual && (obj instanceof AddressWrapper)) {
            AddressWrapper other = (AddressWrapper) obj;
            if (this.addr.length == other.addr.length) {
                isEqual = true;
                for (int i = 0; isEqual && (i < other.addr.length); ++i) {
                    isEqual = (this.addr[i] == other.addr[i]);
                }
            }
        }
        return isEqual;
    }
}
