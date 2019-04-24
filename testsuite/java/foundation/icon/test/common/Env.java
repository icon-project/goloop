package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;

import java.io.FileInputStream;
import java.io.IOException;
import java.util.*;

import static org.junit.Assert.assertNotNull;

public class Env {
    public static final Log LOG = Log.getGlobal();
    public static Node []nodes;
    public static int testApiVer = 3;

    public static class Node {
        private String url;
        public Channel []channels;
        Node(String url) {
            this.url = url;
        }
    }

    public static class Chain {
        public int networkId;
        List<Channel> channelList;
        public KeyWallet godWallet;
        public KeyWallet governorWallet;
        // 0 : init, 1 : enable, -1 : disable
        int audit;
        Chain(int networkId, KeyWallet god, KeyWallet governor, boolean audit) {
            this.networkId = networkId;
            this.godWallet = god;
            this.governorWallet = governor;
            this.audit = 0;
        }
    }

    public static class Channel {
        public Node node;
        public String name;
        public Chain chain;

        Channel(Node node, String name, Chain chain) {
            this.node = node;
            this.name = name;
            this.chain = chain;
        }

        public String getAPIUrl(int v) {
            // TODO apply name for channel later
//            return node.url + "/api/v" + v + "/" + name;
            return node.url + "/api/v" + v;
        }
    }

    private Map<String,Chain> readChains(Properties props) {
        Map<String, Chain> chainMap = new HashMap<>();
        for(int i = 0; ; i++) {
            String chainName = "chain" + i;

            String nid = props.getProperty(chainName + ".nid");
            if (nid == null) {
                break;
            }
            String godWalletPath = props.getProperty(chainName + ".godWallet");
            String godPassword = props.getProperty(chainName + ".godPassword");
            KeyWallet godWallet = null;
            try {
                godWallet = Utils.readWalletFromFile(godWalletPath, godPassword);
            }
            catch (IOException ex) {
                System.out.println("FAIL to read god wallet. path = " + godWalletPath);
                ex.printStackTrace();
            }
            String govWalletPath = props.getProperty(chainName + ".govWallet");
            String govPassword = props.getProperty(chainName + ".govPassword");
            KeyWallet governorWallet = null;
            if(govWalletPath == null) {
                try {
                    governorWallet = KeyWallet.create();
                }
                catch(Exception ex) {
                    System.out.println("FAIL to create wallet for governor!");
                    ex.printStackTrace();
                }
            }
            else {
                try {
                    Utils.readWalletFromFile(govWalletPath, govPassword);
                }
                catch (IOException ex) {
                    System.out.println("FAIL to read governor wallet. path = " + govWalletPath);
                    ex.printStackTrace();
                }
            }
            String audit = props.getProperty(chainName + ".audit");
            boolean bAudit = false;
            if ( audit != null ) {
                bAudit = Boolean.parseBoolean(audit);
            }
            Chain chain = new Chain(Integer.parseInt(nid), godWallet, governorWallet, bAudit);
            chainMap.put(nid, chain);
        }
        return chainMap;
    }

    private List<Node> readNodes(Properties props, Map<String, Chain> chainMap) {
        List<Node> list = new LinkedList<>();
        for( int i = 0; ; i++ ) {
            String nodeName = "node" + i;
            String url = props.getProperty(nodeName + ".url");
            if( url == null ) {
                if(i == 0) {
                    System.out.println("FAIL. no node url");
                    throw new IllegalStateException("FAIL. no node url");
                }
                break;
            }
            Node node = new Node(url);
            // read channel env
            List<Channel> channelList = new LinkedList<>();
            for( int j = 0; ; j++ ) {
                String channelName = nodeName + ".channel" + j;
                String nid = props.getProperty(channelName + ".nid");
                if( nid == null ) {
                    if(j == 0) {
                        System.out.println("FAIL. no nid for channel");
                        throw new IllegalStateException("FAIL. no nid for channel");
                    }
                    break;
                }
                Chain chain = chainMap.get(nid);
                if(chain == null) {
                    throw new IllegalStateException("FAIL. no chain for the " + nid);
                }
                String name = props.getProperty(channelName + ".name", "default");
                channelList.add(new Channel(node, name, chain));
            }
            node.channels = channelList.toArray(new Channel[channelList.size()]);
            list.add(node);
        }
        return list;
    }

    private Env() {
        String env_file = System.getProperty("CHAIN_ENV",
                "./data/env.properties");
        Properties props = new Properties();
        try {
            FileInputStream fi = new FileInputStream(env_file);
            props.load(fi);
            fi.close();
        } catch (IOException e) {
            e.printStackTrace();
            throw new IllegalStateException("There is no environment file name=" + env_file);
        }
        Map<String, Chain> chainMap = readChains(props);
        assertNotNull(chainMap);
        List<Node> nodeList = readNodes(props, chainMap);
        assertNotNull(nodeList);
        Env.nodes = nodeList.toArray(new Node[nodeList.size()]);
    }

    private static class LazyHolder {
        static final Env INSTANCE = new Env();
    }

    public static Env getInstance() {
        return LazyHolder.INSTANCE;
    }
}
