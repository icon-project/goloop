package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;
import java.math.BigInteger;

public class Env {
    public static final Log LOG = Log.getGlobal();

    private class ChannelEnv {
        private final String godPath;
        private final String governorPath;
        private final BigInteger networkId;

        ChannelEnv(String god, BigInteger networkId, String govenor) {
            this.godPath = god;
            if (networkId == null) {
                this.networkId = Constants.DEFAULT_NID;
            }
            else {
                this.networkId = networkId;
            }
            this.governorPath = govenor;
        }

        KeyWallet getGodWallet() {
            KeyWallet wallet = null;
            try {
                if (godPath == null) {
                    wallet = Utils.readWalletFromFile("./data/keystore_god.json", "gochain");
                } else {
                    wallet = Utils.readWalletFromFile(godPath, "gochain");
                }
            }
            catch (IOException ex){
                ex.printStackTrace();
            }
            return wallet;
        }

        BigInteger getNetworkId() {
            return networkId;
        }

        KeyWallet getGovernorWallet() {
            KeyWallet wallet = null;
            try {
                if (governorPath == null) {
                    wallet = KeyWallet.create();
                } else {
                    wallet = Utils.readWalletFromFile(governorPath, "governor");
                }
            }
            catch (Exception ex) {
                ex.printStackTrace();
            }
            return wallet;
        }
    }

    public static class Node {
        public final String endpointUrl;
        public final Chain[] chains;

        public Node(String endpointUrl, ChannelEnv[] env) {
            this.endpointUrl = endpointUrl;

//            assert(env is null)
            this.chains = new Chain[env.length];
            for(int i = 0; i < env.length; i++) {
                ChannelEnv chEnv = env[i];
                Chain chain = new Chain(chEnv.getGodWallet(), chEnv.getNetworkId(), chEnv.getGovernorWallet());
                this.chains[i] = chain;
            }
        }
    }

    public static class Chain {
        public KeyWallet godWallet;
        public KeyWallet governorWallet;
        public BigInteger networkId;
        // 0 : init, 1 : enable, -1 : disable
        public int audit;

        public Chain(KeyWallet god) {
            this.networkId = Constants.DEFAULT_NID;
            this.godWallet = god;
            this.audit = 0;
        }

        public Chain(KeyWallet god, BigInteger nid, KeyWallet governor) {
            godWallet = god;
            networkId = nid;
            governorWallet = governor;
        }

        public boolean isAudit() {
            if ( audit == 0 ) {
                // check audit
                // 0 : init, 1 : enable, -1 : disable
            }
            return this.audit > 0;
        }
    }

    public static Node[] nodes;

    private Env() {
        // pass node & chain environment
        String godPath = System.getProperty("godKey");

        String governorPath = System.getProperty("governorKey");

        String endPath = System.getProperty("endpointUrls");
        nodes = new Node[1];
        if(endPath == null) {
            ChannelEnv[]envs = new ChannelEnv[1];
            envs[0] = new ChannelEnv(godPath, Constants.DEFAULT_NID, governorPath);
            nodes[0] = new Node("http://localhost:9080/api/v3", envs);
        }
        else {
            try {
                BufferedReader reader = new BufferedReader(new FileReader(endPath));
                String endpoint;
                ChannelEnv[]envs = new ChannelEnv[1];
                envs[0] = new ChannelEnv(godPath, Constants.DEFAULT_NID, governorPath);
                int index = 0;
                while ((endpoint = reader.readLine()) != null) {
                    nodes[index] = new Node(endpoint, envs);
                    index++;
                }
                reader.close();
            } catch (IOException ex) {
                System.out.println("Failed to get endpoint. path = " + endPath);
                ex.printStackTrace();
            }
        }
    }

    private static class LazyHolder {
        public static final Env INSTANCE = new Env();
    }

    public static Env getInstance() {
        return LazyHolder.INSTANCE;
    }
}
