package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;
import java.math.BigInteger;
import java.util.LinkedList;
import java.util.List;

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

        public Chain(KeyWallet god) {
            this.networkId = Constants.DEFAULT_NID;
            this.godWallet = god;
        }

        public Chain(KeyWallet god, BigInteger nid) {
            this.networkId = nid;
            this.godWallet = god;
        }
    }

    private static KeyWallet godWallet;
    private static List<String> endPoints;

    public static KeyWallet getGodWallet() {
        if (godWallet == null) {
            String path = System.getProperty("godKey");
            if (path == null) {
                return null;
            }
            try {
                godWallet = Utils.readWalletFromFile(path, "gochain");
            } catch (IOException ex) {
                ex.printStackTrace();
            }
        }
        return godWallet;
    }

    public static List<String> getEndpoint() {
        if (endPoints == null) {
            String path = System.getProperty("endpointUrls");
            if (path == null) {
                return null;
            }
            List<String> list = new LinkedList<>();
            try {
                BufferedReader reader = new BufferedReader(new FileReader(path));
                String endpoint;
                while ((endpoint = reader.readLine()) != null) {
                    list.add(endpoint);
                }
                reader.close();
                endPoints = list;
            } catch (IOException ex) {
                System.out.println("Failed to get endpoint. path = " + path);
                ex.printStackTrace();
            }
        }
        return endPoints;
    }
}
