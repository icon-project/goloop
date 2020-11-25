package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Method;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;

import java.io.IOException;
import java.math.BigInteger;

public class Contract {
    private final ServiceManager sm;
    private final Address address;
    private final Method[] methods;

    public Contract(ServiceManager sm, Address address, Method[] methods) {
        this.sm = sm;
        this.address = address;
        this.methods = methods;
    }

    Result invoke(
            boolean query, boolean must, BigInteger value,
            BigInteger stepLimit, String method, Object[] params
    ) {
        try {
            Method m = getMethod(method);
            if (m == null) {
                throw new TransactionException(new Result(
                        Status.MethodNotFound,
                        BigInteger.ZERO,
                        "Method not found: " + method));
            }
            if (query && (m.getFlags() & Method.Flags.READONLY) == 0) {
                throw new TransactionException(new Result(
                        Status.AccessDenied,
                        BigInteger.ZERO,
                        "Method not found"));
            }
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

    public Method[] getMethods() {
        return methods;
    }

    public Method getMethod(String name) {
        for (var m : methods) {
            if (m.getName().equals(name)) {
                return m;
            }
        }
        return null;
    }
}
