package foundation.icon.ee.test;

import avm.Address;

public class Contract {
    private SMProxy sm;
    private Address address;

    public Contract(SMProxy sm, Address address) {
        this.sm = sm;
        this.address = address;
    }

    public Result invoke(String method, Object... params) {
        try {
            return sm.invoke(address, sm.getValue(), sm.getStepLimit(), method, params);
        } catch (Exception e) {
            throw new AssertionError(e);
        }
    }

    public Address getAddress() {
        return address;
    }
}
