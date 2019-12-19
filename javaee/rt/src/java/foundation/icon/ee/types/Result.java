package foundation.icon.ee.types;

import java.math.BigInteger;

public class Result {
    private final int status;
    private final BigInteger stepUsed;
    private final Object ret;

    public Result(int status, BigInteger stepUsed, Object ret) {
        this.status = status;
        this.stepUsed = stepUsed;
        this.ret = ret;
    }

    public int getStatus() {
        return status;
    }

    public BigInteger getStepUsed() {
        return stepUsed;
    }

    public Object getRet() {
        return ret;
    }
}
