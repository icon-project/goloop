package foundation.icon.test.common;


import foundation.icon.icx.data.Bytes;

public class ResultTimeoutException extends Exception{
    Bytes txHash;
    public ResultTimeoutException(Bytes txHash) {
        this.txHash = txHash;
    }

    public Bytes getTxHash() {
        return this.txHash;
    }
}
