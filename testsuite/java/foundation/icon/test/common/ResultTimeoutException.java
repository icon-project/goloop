package foundation.icon.test.common;

import foundation.icon.icx.data.Bytes;

public class ResultTimeoutException extends Exception{
    Bytes txHash;

    public ResultTimeoutException() {
        super();
    }

    public ResultTimeoutException(String message) {
        super(message);
    }

    public ResultTimeoutException(Bytes txHash) {
        super("Timeout. txHash=" + txHash);
        this.txHash = txHash;
    }

    public Bytes getTxHash() {
        return this.txHash;
    }
}
