package foundation.icon.ee.test;

import score.Address;

import java.io.IOException;
import java.math.BigInteger;

public class Contract {
    private final ServiceManager sm;
    private final Address address;

    public Contract(ServiceManager sm, Address address) {
        this.sm = sm;
        this.address = address;
    }

    Result invoke(
            boolean query, boolean must, Address address, BigInteger value,
            BigInteger stepLimit, String method, Object[] params
    ) {
        try {
            var res =  sm.invoke(query, address, value, stepLimit, method, params);
            if (must && res.getStatus() != 0) {
                throw new TransactionException(res);
            }
            return res;
        } catch (IOException e) {
            throw new AssertionError(e);
        }
    }

    public Result invoke(String method, Object... params) {
        return invoke(false, true, address, sm.getValue(),
                sm.getStepLimit(), method, params);
    }

    public Result invoke(BigInteger value, BigInteger stepLimit, String method,
                         Object... params) {
        return invoke(false, true, address, value, stepLimit, method,
                params);
    }

    public Result tryInvoke(String method, Object... params) {
        return invoke(false, false, address, sm.getValue(),
                sm.getStepLimit(), method, params);
    }

    public Result tryInvoke(BigInteger value, BigInteger stepLimit, String method,
                         Object... params) {
        return invoke(false, false, address, value, stepLimit, method,
                params);
    }

    public Result query(String method, Object... params) {
        return invoke(true, true, address, sm.getValue(),
                sm.getStepLimit(), method, params);
    }

    public Address getAddress() {
        return address;
    }
}
