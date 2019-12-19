package foundation.icon.ee.test;

import java.math.BigInteger;

public class Result {
    private int status;
    private BigInteger stepUsed;
    private Object result;

    public Result(int status, BigInteger stepUsed, Object result) {
        this.status = status;
        this.stepUsed = stepUsed;
        this.result = result;
    }

    public int getStatus() {
        return status;
    }

    public BigInteger getStepUsed() {
        return stepUsed;
    }

    public Object getResult() {
        return result;
    }
}
