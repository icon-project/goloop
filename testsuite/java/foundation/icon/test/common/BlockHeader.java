package foundation.icon.test.common;

import foundation.icon.ee.types.Address;

public class BlockHeader {
    private final byte[] bytes;
    private final int version;
    private final long height;
    private final Address proposer;
    private final byte[] prevID;
    private final long timestamp;
    private final byte[] votesHash;
    private final byte[] nextValidatorHash;
    private final byte[] normalReceiptHash;

    public BlockHeader(byte[] bytes, Codec codec) {
        this.bytes = bytes;
        var r = codec.newReader(bytes);
        r.readListHeader();

        version = r.readInt();
        height = r.readLong();
        timestamp = r.readLong();
        proposer = new Address(r.readByteArray());
        if (r.readNullity()) {
            prevID = null;
        } else {
            prevID = r.readByteArray();
        }
        if (r.readNullity()) {
            votesHash = null;
        } else {
            votesHash = r.readByteArray();
        }
        nextValidatorHash = r.readByteArray();
        r.skip(3); // PatchTransactionsHash, NormalTransactionHash, LogBloom

        var rr = codec.newReader(r.readByteArray());
        rr.readListHeader();
        rr.skip(2); // StateHash, PatchReceiptsHash
        if (!rr.readNullity()) {
            normalReceiptHash = rr.readByteArray();
        } else {
            normalReceiptHash = null;
        }
        while (rr.hasNext()) {
            rr.skip(1);
        }
        rr.readFooter();

        // clean-up remains
        while (r.hasNext()) {
            r.skip(1);
        }
        r.readFooter();
    }

    public byte[] getBytes() {
        return bytes;
    }

    public int getVersion() {
        return version;
    }

    public long getHeight() {
        return height;
    }

    public byte[] getVotesHash() {
        return votesHash;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public byte[] getNextValidatorHash() {
        return nextValidatorHash;
    }

    public byte[] getPrevID() {
        return prevID;
    }

    public byte[] getNormalReceiptHash() {
        return normalReceiptHash;
    }
}
