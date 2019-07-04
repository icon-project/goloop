package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.EventGen;
import org.java_websocket.client.WebSocketClient;
import org.java_websocket.drafts.Draft_6455;
import org.java_websocket.handshake.ServerHandshake;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.net.URI;
import java.net.URISyntaxException;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertTrue;

// TODO What about adding annotation indicating requirements. For example,
// "@require(nodeNum=4,chainNum=1)" indicates it requires at least 4 nodes and
// 1 chain for each.
@Tag(Constants.TAG_NORMAL)
public class WSEventTest {
    private static Env.Chain chain;
    private static IconService iconService;
    private static String apiURLBase;

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        apiURLBase = channel.getWSAPIUrl(Env.testApiVer) + "/";
    }

    private String recvBuffer;
    private Object condVar = new Object();

    private void notifyMessage(String msg) {
        synchronized (condVar) {
            recvBuffer = msg;
            condVar.notify();
        }
    }

    private boolean waitForMessage() {
        synchronized (condVar) {
            while (recvBuffer == null) {
                try {
                    condVar.wait();
                } catch (InterruptedException e) {
                    LOG.severe("Unexpected interrupt" + e);
                    return false;
                }
            }
        }
        return true;
    }

    private WebSocketClient newWSClient(String path) {
        URI uri;
        try {
            LOG.info("connect to " + apiURLBase + path);
            uri = new URI(apiURLBase + path);
        } catch (URISyntaxException e) {
            LOG.severe("URISyntaxException " + e);
            e.printStackTrace();
            return null;
        }
        WebSocketClient cc;
        cc = new WebSocketClient(uri, new Draft_6455()) {
            @Override
            public void onMessage(String message) {
                LOG.info("onMessage " + message);
                notifyMessage(message);
            }

            @Override
            public void onOpen(ServerHandshake handshake) {
                LOG.info("onOpen");
                String request = String.join("\n",
                        "{",
                        "\"height\": \"0\",",
                        "\"event\" : \"Event(Address,int,bytes)\"",
                        "}");
                LOG.info("send :" + request);
                send(request);
            }

            @Override
            public void onClose(int code, String reason, boolean remote) {
                LOG.info("onClose code=" + code + " reason=" + reason + " remote=" + remote);
            }

            @Override
            public void onError(Exception ex) {
                LOG.info("onError " + ex);
                ex.printStackTrace();
            }
        };

        cc.connect();
        return cc;
    }

    @Test
    public void wsEventTest() throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();

        LOG.infoEntering("transfer", "initial icx to owner address");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, ownerWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, ownerWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        LOG.infoEntering("deploy", "event gen SCORE");
        EventGen eventGen = EventGen.install(iconService, chain, ownerWallet, 18);
        LOG.infoExiting();

        WebSocketClient cc = newWSClient("event");
        LOG.infoEntering("wait", "response");
        assertTrue(waitForMessage());
        LOG.infoExiting();

        recvBuffer = null;
        LOG.infoEntering("invoke", "generate");
        eventGen.invokeGenerate(aliceWallet, bobWallet.getAddress(), new BigInteger("100"), new byte[]{1});
        LOG.infoExiting();
        LOG.infoEntering("wait", "message");
        assertTrue(waitForMessage());
        LOG.infoExiting();

        recvBuffer = null;
        LOG.infoEntering("invoke", "generate");
        eventGen.invokeGenerate(aliceWallet, bobWallet.getAddress(), new BigInteger("100"), new byte[]{2});
        LOG.infoExiting();
        LOG.infoEntering("wait", "message");
        assertTrue(waitForMessage());
        LOG.infoExiting();
    }
}
