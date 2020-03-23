package foundation.icon.test.common;

import foundation.icon.ee.types.Address;

import java.util.ArrayList;

public class ValidatorList {
    private final byte[] bytes;
    private final Address[] validators;

    public ValidatorList(byte[] bytes, Codec c) {
        this.bytes = bytes;
        var r = c.newReader(bytes);
        r.readListHeader();
        var vl = new ArrayList<Address>();
        while (r.hasNext()) {
            var addr = new Address(r.readByteArray());
            vl.add(addr);
        }
        validators = vl.toArray(new Address[0]);
        r.readFooter();
    }

    public byte[] getBytes() {
        return bytes;
    }

    public boolean contains(Address addr) {
        for (Address validator : validators) {
            if (validator.equals(addr)) {
                return true;
            }
        }
        return false;
    }

    public int indexOf(Address addr) {
        for (int i = 0; i < validators.length; i++) {
            if (validators[i].equals(addr)) {
                return i;
            }
        }
        return -1;
    }

    public int size() {
        return validators.length;
    }
}
