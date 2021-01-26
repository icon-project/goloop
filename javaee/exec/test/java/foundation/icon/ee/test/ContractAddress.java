package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;

import java.io.IOException;
import java.math.BigInteger;

public class ContractAddress {
    private final ServiceManager sm;
    private final Address address;

    public ContractAddress(ServiceManager sm, Address address) {
        this.sm = sm;
        this.address = address;
    }

    Result invoke(
            boolean query, boolean must, BigInteger value,
            BigInteger stepLimit, String method, Object[] params
    ) {
        try {
            var res =  sm.invoke(query, address, value, stepLimit, method, params);
            if (must && res.getStatus() != Status.Success) {
                throw new TransactionException(res);
            }
            return res;
        } catch (IOException e) {
            throw new AssertionError(e);
        }
    }

    public Result invoke(String method, Object... params) {
        return invoke(false, true, sm.getValue(), sm.getStepLimit(),
                method, params);
    }

    public Result invoke(BigInteger value, BigInteger stepLimit, String method,
                         Object... params) {
        return invoke(false, true, value, stepLimit, method, params);
    }

    public Result tryInvoke(String method, Object... params) {
        return invoke(false, false, sm.getValue(),
                sm.getStepLimit(), method, params);
    }

    public Result tryInvoke(BigInteger value, BigInteger stepLimit, String method,
                         Object... params) {
        return invoke(false, false, value, stepLimit, method,
                params);
    }

    public Result query(String method, Object... params) {
        return invoke(true, true, sm.getValue(), sm.getStepLimit(),
                method, params);
    }

    public Address getAddress() {
        return address;
    }
}
