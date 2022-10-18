/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.crypto.KeystoreException;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.Properties;

import static org.junit.jupiter.api.Assertions.assertNotNull;

public class Env {
    public static final Log LOG = Log.getGlobal();
    public static Node[] nodes;
    public static Chain[] chains;
    public static int testApiVer = 3;
    private static String dataPath;

    public static class Node {
        private final String url;
        public final KeyWallet wallet;
        public Channel[] channels;

        Node(String url, KeyWallet wallet) {
            this.url = url;
            this.wallet = wallet;
        }
    }

    public static class Chain {
        private final Properties props;
        private final String prefix;
        private List<Channel> channelList;

        public final int networkId;
        public Channel[] channels;
        public final KeyWallet godWallet;
        public final KeyWallet governorWallet;

        Chain(Properties props, String prefix, int networkId, KeyWallet god, KeyWallet governor) {
            this.props = props;
            this.prefix = prefix;
            this.networkId = networkId;
            this.godWallet = god;
            this.governorWallet = governor;
            this.channelList = new LinkedList<>();
        }

        public String getProperty(String key) {
            return this.props.getProperty(prefix + key);
        }

        public String getProperty(String key, String def) {
            return this.props.getProperty(prefix+key, def);
        }

        private void makeChannels() {
            channels = channelList.toArray(new Channel[0]);
            channelList = null;
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
            return node.url + "/api/v" + v + "/" + name;
        }

        public String getWSAPIUrl(int v) {
            String api = getAPIUrl(v);
            if (api.startsWith("http:")) {
                return "ws:" + api.substring(5);
            } else if (api.startsWith("https:")) {
                return "wss:" + api.substring(6);
            }
            return api;
        }
    }

    private static int parseInt(String s) {
        if (s.startsWith("0x")) {
            return Integer.parseInt(s.substring(2), 16);
        } else if (s.startsWith("0") && s.length() > 1) {
            return Integer.parseInt(s.substring(1), 8);
        } else {
            return Integer.parseInt(s);
        }
    }

    private static KeyWallet loadWallet(String path, String password) throws IOException, KeystoreException {
        Path walletPath = Path.of(path);
        if (!walletPath.isAbsolute()) {
            walletPath = Path.of(dataPath, path).toAbsolutePath();
        }
        return KeyWallet.load(password, walletPath.toFile());
    }

    private static Map<String, Chain> readChains(Properties props) {
        Map<String, Chain> chainMap = new HashMap<>();
        for (int i = 0; ; i++) {
            String chainName = "chain" + i;

            String nid = props.getProperty(chainName + ".nid");
            if (nid == null) {
                if (i == 0) {
                    System.out.println("FAIL. no nid for chain");
                    throw new IllegalStateException("FAIL. no nid for channel");
                }
                break;
            }
            String godWalletPath = props.getProperty(chainName + ".godWallet");
            KeyWallet godWallet = null;
            try {
                godWallet = loadWallet(godWalletPath, props.getProperty(chainName + ".godPassword"));
            } catch (IOException | KeystoreException e) {
                e.printStackTrace();
                String message = String.format("FAIL to load god wallet, path: %s, err: %s", godWalletPath, e.getMessage());
                System.out.println(message);
                throw new IllegalArgumentException(message);
            }
            String govWalletPath = props.getProperty(chainName + ".govWallet");
            KeyWallet governorWallet = null;
            if (govWalletPath == null) {
                try {
                    governorWallet = KeyWallet.create();
                } catch (Exception e) {
                    String message = String.format("FAIL to create governor wallet, err: %s", e.getMessage());
                    System.out.println(message);
                    throw new IllegalArgumentException(message);
                }
            } else {
                try {
                    governorWallet = loadWallet(govWalletPath, props.getProperty(chainName + ".govPassword"));
                } catch (IOException | KeystoreException e) {
                    e.printStackTrace();
                    String message = String.format("FAIL to load governor wallet, path: %s, err: %s", govWalletPath, e.getMessage());
                    System.out.println(message);
                    throw new IllegalArgumentException(message);
                }
            }
            Chain chain = new Chain(props, chainName + ".", parseInt(nid), godWallet, governorWallet);
            chainMap.put(nid, chain);
        }
        return chainMap;
    }

    private static List<Node> readNodes(Properties props, Map<String, Chain> chainMap) {
        List<Node> nodeList = new LinkedList<>();
        for (int i = 0; ; i++) {
            String nodeName = "node" + i;
            String url = props.getProperty(nodeName + ".url");
            if (url == null) {
                if (i == 0) {
                    System.out.println("FAIL. no node url");
                    throw new IllegalStateException("FAIL. no node url");
                }
                break;
            }
            String nodeWalletPath = props.getProperty(nodeName + ".wallet");
            KeyWallet nodeWallet = null;
            if (nodeWalletPath != null) {
                try {
                    nodeWallet = loadWallet(nodeWalletPath, props.getProperty(nodeName + ".walletPassword"));
                } catch (IOException | KeystoreException e) {
                    e.printStackTrace();
                    String message = String.format("FAIL to load node wallet, path: %s, err: %s", nodeWalletPath, e.getMessage());
                    System.out.println(message);
                    throw new IllegalArgumentException(message);
                }
            }
            Node node = new Node(url, nodeWallet);

            // read channel env
            List<Channel> channelsOnNode = new LinkedList<>();
            for (int j = 0; ; j++) {
                String channelName = nodeName + ".channel" + j;
                String nid = props.getProperty(channelName + ".nid");
                if (nid == null) {
                    if (j == 0) {
                        System.out.println("FAIL. no nid for channel");
                        throw new IllegalArgumentException("FAIL. no nid for channel");
                    }
                    break;
                }
                Chain chain = chainMap.get(nid);
                if (chain == null) {
                    System.out.println("FAIL. no chain for the " + nid);
                    throw new IllegalStateException("FAIL. no chain for the " + nid);
                }
                String name = props.getProperty(channelName + ".name", "default");
                Channel channel = new Channel(node, name, chain);
                channelsOnNode.add(channel);
                chain.channelList.add(channel);
            }
            node.channels = channelsOnNode.toArray(new Channel[0]);
            nodeList.add(node);
        }
        for (Chain chain : chainMap.values()) {
            chain.makeChannels();
        }
        return nodeList;
    }

    static {
        String env_file = System.getProperty("CHAIN_ENV", "./data/env.properties");
        dataPath = Paths.get("data").toAbsolutePath().toString() + "/";
        Properties props = new Properties();
        try {
            System.out.println("Current env.properties: " + env_file);
            FileInputStream fi = new FileInputStream(env_file);
            props.load(fi);
            fi.close();
        } catch (IOException e) {
            System.out.println("There is no environment file name=" + env_file);
            throw new IllegalStateException("There is no environment file name=" + env_file);
        }
        Map<String, Chain> chainMap = readChains(props);
        assertNotNull(chainMap);
        Env.chains = chainMap.values().toArray(new Chain[0]);

        List<Node> nodeList = readNodes(props, chainMap);
        assertNotNull(nodeList);
        Env.nodes = nodeList.toArray(new Node[0]);
    }
}
