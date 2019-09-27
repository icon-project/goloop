package org.aion.kernel;

import java.math.BigInteger;

import org.aion.types.AionAddress;


public final class TestingBlock {

    private final byte[] prevHash;
    private final long number;
    private final AionAddress coinbase;
    private final long timestamp;
    private final byte[] data;
    private final long energyLimit;
    private final BigInteger difficulty;


    public TestingBlock(byte[] prevHash, long number, AionAddress coinbase, long timestamp, byte[] data) {
        this.prevHash = prevHash;
        this.number = number;
        this.coinbase = coinbase;
        this.timestamp = timestamp;
        this.data = data;
        this.energyLimit = 10_000_000L;
        this.difficulty = BigInteger.valueOf(10_000_000L);
    }

    public byte[] getPrevHash() {
        return prevHash;
    }

    public long getNumber() {
        return number;
    }

    public AionAddress getCoinbase() {
        return coinbase;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public byte[] getData() {
        return data;
    }

    public long getEnergyLimit() {
        return energyLimit;
    }

    public BigInteger getDifficulty() {
        return difficulty;
    }
}
