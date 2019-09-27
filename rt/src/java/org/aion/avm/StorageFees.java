package org.aion.avm;

/**
 * A few constants to describe how we limit and bill for storage graph serialization/deserialization.
 */
public class StorageFees {
    public static final int MAX_GRAPH_SIZE = 500_000;
    public static final int READ_PRICE_PER_BYTE = 1;
    public static final int WRITE_PRICE_PER_BYTE = 3;
}
