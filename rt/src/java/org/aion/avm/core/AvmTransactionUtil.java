package org.aion.avm.core;

import java.math.BigInteger;
import org.aion.types.InternalTransaction;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;

public class AvmTransactionUtil {

    /**
     * Factory method to create a 'call' Transaction.
     */
    public static Transaction call(AionAddress sender, AionAddress destination, BigInteger nonce, BigInteger value, byte[] data, String method, Object[] params, long energyLimit, long energyPrice) {
        return Transaction.contractCallTransaction(sender
            , destination
            , new byte[32]
            , nonce
            , value
            , data
            , method
            , params
            , energyLimit
            , energyPrice
        );
    }

    /**
     * Factory method to create a 'create' Transaction.
     */
    public static Transaction create(AionAddress sender, BigInteger nonce, BigInteger value, byte[] data, String method, Object[] params, long energyLimit, long energyPrice) {
        return Transaction.contractCreateTransaction(sender
            , new byte[32]
            , nonce
            , value
            , data
            , method
            , params
            , energyLimit
            , energyPrice
        );
    }

    /**
     * Factory method to create the Transaction data type from an InternalTransaction.
     *
     * @param internalTransaction The transaction we were given.
     * @return The new Transaction instance.
     * @throws IllegalArgumentException If any elements of external are statically invalid.
     */
    public static Transaction fromInternalTransaction(InternalTransaction internalTransaction) {
        // TODO: pass method and params
        if (internalTransaction.isCreate) {
            return Transaction.contractCreateTransaction(internalTransaction.sender
                    , new byte[32]
                    , internalTransaction.senderNonce
                    , internalTransaction.value
                    , internalTransaction.copyOfData()
                    , null
                    , null
                    , internalTransaction.energyLimit
                    , internalTransaction.energyPrice
            );
        } else {
            return Transaction.contractCallTransaction(internalTransaction.sender
                    , internalTransaction.destination
                    , new byte[32]
                    , internalTransaction.senderNonce
                    , internalTransaction.value
                    , internalTransaction.copyOfData()
                    , null
                    , null
                    , internalTransaction.energyLimit
                    , internalTransaction.energyPrice
            );
        }
    }
}
