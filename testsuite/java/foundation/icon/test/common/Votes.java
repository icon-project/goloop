package foundation.icon.test.common;

import foundation.icon.ee.io.DataReader;
import foundation.icon.ee.io.DataWriter;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.util.Crypto;

import java.util.ArrayList;
import java.util.Base64;

public class Votes {
    private final byte[] bytes;
    private final long round;
    private final PartSetID blockPartSetID;
    private final VoteItem[] voteItems;

    public static class VoteItem {
        private final long timestamp;
        private final byte[] signature;

        public VoteItem(byte[] bytes, Codec c) {
            this(c.newReader(bytes));
        }

        public VoteItem(DataReader r) {
            r.readListHeader();
            timestamp = r.readLong();
            signature = r.readByteArray();
            while (r.hasNext()) {
                r.skip(1);
            }
            r.readFooter();
        }

        public long getTimestamp() {
            return timestamp;
        }

        public byte[] getSignature() {
            return signature;
        }
    }

    public static class PartSetID {
        private final int count;
        private final byte[] hash;

        public PartSetID(DataReader r) {
            r.readListHeader();
            count = r.readInt();
            hash = r.readByteArray();
            while (r.hasNext()) {
                r.skip(1);
            }
            r.readFooter();
        }

        public int getCount() {
            return count;
        }

        public byte[] getHash() {
            return hash;
        }

        public void writeTo(DataWriter w) {
            w.writeListHeader(2);
            w.write(count);
            w.write(hash);
            w.writeFooter();
        }
    }

    public Votes(byte[] bytes, Codec c) {
        this.bytes = bytes;
        var r = c.newReader(bytes);
        r.readListHeader();

        round = r.readLong();
        blockPartSetID = new PartSetID(r);

        r.readListHeader();
        var items = new ArrayList<VoteItem>();
        while (r.hasNext()) {
            var item = new VoteItem(r);
            items.add(item);
        }
        r.readFooter();
        voteItems = items.toArray(new VoteItem[0]);

        while (r.hasNext()) {
            r.skip(1);
        }
        r.readFooter();
    }

    public byte[] getBytes() {
        return bytes;
    }

    public long getRound() {
        return round;
    }

    public int verifyVotes(BlockHeader blk, ValidatorList validators, Codec c) {
        var id = blk.getPrevID();
        var height = blk.getHeight()-1;
        int verified = 0;
        boolean[] checked = new boolean[validators.size()];
        for (VoteItem item : voteItems) {
            var w = c.newWriter();
            w.writeListHeader(6);
            w.write(height);
            w.write(round);
            w.write(1);
            w.write(id);
            blockPartSetID.writeTo(w);
            w.write(item.getTimestamp());
            w.writeFooter();
            var voteMsg = w.toByteArray();
            byte[] msgHash = Crypto.sha3_256(voteMsg);
            byte[] pubKey = Crypto.recoverKey(msgHash, item.getSignature(), false);
            if (pubKey == null) {
                throw new IllegalArgumentException("FailToRecoverFromTheSignature");
            }
            var addr = new Address(Crypto.getAddressBytesFromKey(pubKey));
            var idx = validators.indexOf(addr);
            if (idx < 0) {
                var encoder = Base64.getEncoder();
                System.err.println("VoteMsg: " + encoder.encodeToString(voteMsg));
                System.err.println("Signature: " + encoder.encodeToString(item.getSignature()));
                System.err.println("Address: " + addr);
                continue;
            }
            if (checked[idx]) {
                continue;
            }
            checked[idx] = true;
            verified++;
        }
        return verified;
    }
}
