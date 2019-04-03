package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;

import java.math.BigInteger;

public class Env {
    public static final Log LOG = Log.getGlobal();

    public static Node[] nodes;

    public static class Node {
        public final String endpointUrl;
        public final Chain[] chains;

        public Node(String endpointUrl, Chain[] chains) {
            this.endpointUrl = endpointUrl;
            this.chains = chains;
        }
    }

    public static class Chain {
        public final BigInteger networkId;
        public final KeyWallet godWallet;

        public Chain(BigInteger nid, KeyWallet god) {
            this.networkId = nid;
            this.godWallet = god;
        }
    }
}
