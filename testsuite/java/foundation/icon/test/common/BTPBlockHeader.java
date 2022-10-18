package foundation.icon.test.common;

import foundation.icon.ee.io.*;

import java.util.*;

public class BTPBlockHeader {
    private final byte[] bytes;
    private final long mainHeight;
    private final int round;
    private final byte[] nextProofContextHash;
    private final MerkleNode[] networkSectionToRoot;
    private final long networkID;
    private final long updateNumber;
    private final byte[] prev;
    private final int messageCount;
    private final byte[] messageRoot;
    private final byte[] nextProofContext;

    public static class MerkleNode {
        private final int dir;
        private final byte[] value;

        public MerkleNode(DataReader r) {
            r.readListHeader();
            dir = r.readInt();
            value = r.readByteArray();
            while (r.hasNext()) {
                r.skip(1);
            }
            r.readFooter();
        }

        public int getDir() {
            return dir;
        }

        public byte[] getValue() {
            return value;
        }
    }

    public BTPBlockHeader(byte[] bytes, Codec codec) {
        this.bytes = bytes;
        var r = codec.newReader(bytes);
        r.readListHeader();

        mainHeight = r.readLong();
        round = r.readInt();
        nextProofContextHash = r.readByteArray();
        if (r.readNullity()) {
            networkSectionToRoot = null;
        } else {
            r.readListHeader();
            var nodes = new ArrayList<MerkleNode>();
            while (r.hasNext()) {
                var item = new MerkleNode(r);
                nodes.add(item);
            }
            r.readFooter();
            networkSectionToRoot = nodes.toArray(new MerkleNode[0]);
        }
        networkID = r.readLong();
        updateNumber = r.readLong();
        if (r.readNullity()) {
            prev = null;
        } else {
            prev = r.readByteArray();
        }
        messageCount = r.readInt();
        if (r.readNullity()) {
            messageRoot = null;
        } else {
            messageRoot = r.readByteArray();
        }
        if (r.readNullity()) {
            nextProofContext = null;
        } else {
            nextProofContext = r.readByteArray();
        }

        // clean-up remains
        while (r.hasNext()) {
            r.skip(1);
        }
        r.readFooter();
    }

    public byte[] getBytes() {
        return bytes;
    }

    public long getMainHeight() {
        return mainHeight;
    }

    public int getRound() {
        return round;
    }

    public byte[] getNextProofContextHash() {
        return nextProofContextHash;
    }

    public MerkleNode[] getNetworkSectionToRoot() {
        return networkSectionToRoot;
    }

    public long getNetworkID() {
        return networkID;
    }

    public long getUpdateNumber() {
        return updateNumber;
    }

    public long getFirstMessageSN() {
        return updateNumber >> 1;
    }

    public boolean getNextProofContextChanged() {
        return (updateNumber & 0x1) != 0;
    }

    public byte[] getPrev() {
        return prev;
    }

    public int getMessageCount() {
        return messageCount;
    }

    public byte[] getMessageRoot() {
        return messageRoot;
    }

    public byte[] getNextProofContext() {
        return nextProofContext;
    }

}
