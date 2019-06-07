package foundation.icon.test.common;

import foundation.icon.icx.data.Address;

import java.math.BigInteger;

public class NotEnoughBalanceException extends Exception {
    public NotEnoughBalanceException() {
        super();
    }

    public NotEnoughBalanceException(String message) {
        super(message);
    }

    public NotEnoughBalanceException(Address addr, BigInteger balance, BigInteger value) {
        super("Not enough balance. ID(" + addr + "), balance(" + balance + "), value(" + value + ")");
    }
}
