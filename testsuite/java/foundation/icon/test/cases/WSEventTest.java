package foundation.icon.test.cases;

import foundation.icon.icx.*;
import foundation.icon.icx.data.*;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.monitor.Monitor;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.Utils;
import foundation.icon.test.score.EventGen;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.LinkedList;
import java.util.List;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.*;

// TODO What about adding annotation indicating requirements. For example,
// "@require(nodeNum=4,chainNum=1)" indicates it requires at least 4 nodes and
// 1 chain for each.
@Tag(Constants.TAG_NORMAL)
public class WSEventTest {
    private static Env.Chain chain;
    private static IconService iconService;
    private Object condVar = new Object();

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
    }


    /*
    It receives 10 blocks from the last block.
    After that, it confirms whether the received blocks are transferred in order or whether the blocks are normal.
     */
    @Test
    public void wsBlkMonitorTest() throws Exception {
        Block lastBlk = iconService.getLastBlock().execute();
        BigInteger reqBlkHeight = lastBlk.getHeight().add(BigInteger.TEN);
        Monitor<BlockNotification> bm = iconService.monitorBlocks(reqBlkHeight);
        List<BlockNotification> notiList = new LinkedList<>();
        final int testBlkNum = 10;
        bm.start(new Monitor.Listener<BlockNotification>() {
            boolean complete = false;
            boolean stop = false;

            @Override
            public void onStart() {
                synchronized (condVar) {
                    assertFalse(stop);
                }
            }

            @Override
            public void onEvent(BlockNotification event) {
                synchronized (condVar) {
                    assertFalse(stop);
                    if(!complete) {
                        LOG.info("received block " + event.getHeight());
                        notiList.add(event);
                        if(notiList.size() == testBlkNum) {
                            complete = true;
                            condVar.notify();
                        }
                    }
                }
            }

            @Override
            public void onError(long code) {
                throw new RuntimeException("onError code : " + code);
            }

            @Override
            public void onClose() {
                synchronized (condVar) {
                    assertFalse(stop);
                    stop = true;
                    notiList.clear();
                    condVar.notify();
                }
            }
        });

        BlockNotification noti = null;
        BigInteger height = reqBlkHeight;
        synchronized (condVar) {
            condVar.wait(3000 * testBlkNum);
            assertEquals(testBlkNum, notiList.size());
        }
        for(int i = 0; i < testBlkNum; i++) {
            noti = notiList.get(i);
            assertFalse(noti == null);
            LOG.infoEntering("check received block " + noti.getHeight());
            // check the order og the received blocks
            int cmp = noti.getHeight().compareTo(height.add(BigInteger.valueOf(i)));
            assertTrue(cmp == 0);
            Block blk = iconService.getBlock(noti.getHash()).execute();
            cmp = blk.getHeight().compareTo(noti.getHeight());
            assertTrue(cmp == 0);
            LOG.infoExiting();
        }

        LOG.infoEntering("stop");
        bm.stop();
        synchronized (condVar) {
            condVar.wait(3000);
            assertTrue(notiList.size() == 0);
        }
        LOG.infoExiting();
    }

    @Test
    public void wsEvtMonitorTest() throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();

        LOG.infoEntering("transfer", "initial icx to owner address");
        Utils.transferIcx(iconService, chain.networkId, chain.godWallet, ownerWallet.getAddress(), "100");
        Utils.ensureIcxBalance(iconService, ownerWallet.getAddress(), 0, 100);
        LOG.infoExiting();

        // deploy 2 scores with same source
        EventGen eventGen[] = new EventGen[2];
        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[0] = EventGen.install(iconService, chain, ownerWallet, 18);
        LOG.infoExiting();

        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[1] = EventGen.install(iconService, chain, ownerWallet, 18);
        LOG.infoExiting();

        String event = "Event(Address,int,bytes)";
        Address addrs[] = new Address[] {
                null, eventGen[0].getAddress(), eventGen[0].getAddress(),
        };
        String data[][] = new String[][] {
                null,
                null,
                {bobWallet.getAddress().toString(), "500", "0x0A"},
        };
        LOG.info("bobAddr : " + bobWallet.getAddress().toString());
        int expectedEventNum[] = {4,2,1};

        for(int i = 0; i < 3; i++) {
            List<EventNotification> eventList = new LinkedList<>();
            LOG.infoEntering("request monitor[" + i + "]");
            Block lastBlk = iconService.getLastBlock().execute();
            /*
            1. monitor with event
            2. monitor with event and address
            3. monitor with event, address and data
             */
            Monitor<EventNotification> em = iconService.monitorEvents(lastBlk.getHeight(), event, addrs[i], data[i]);
            boolean started = em.start(new Monitor.Listener<EventNotification>() {
                boolean stop = false;

                @Override
                public void onStart() {
                    synchronized (condVar) {
                        assertFalse(stop);
                    }
                }

                @Override
                public void onEvent(EventNotification event) {
                    synchronized (condVar) {
                        assertFalse(stop);
                        LOG.info("receive height : " + event.getHeight() + ", index : " + event.getIndex());
                        eventList.add(event);
                    }
                }

                @Override
                public void onError(long code) {
                    throw new RuntimeException();
                }

                @Override
                public void onClose() {
                    synchronized (condVar) {
                        assertFalse(stop);
                        stop = true;
                    }
                }
            });
            if(!started) {
                throw new IllegalStateException();
            }

            for(EventGen eg : eventGen) {
                Bytes txHash = eg.invokeGenerate(aliceWallet, bobWallet.getAddress(), new BigInteger("100"), new byte[]{1});
                LOG.info("sendTx : " + txHash);
            }

            for(EventGen eg : eventGen) {
                TransactionResult txResult = eg.invokeGenerateAndWait(aliceWallet, bobWallet.getAddress(), new BigInteger(data[2][1]), new byte[]{Byte.decode(data[2][2])});
                LOG.info("sendTx : " + txResult.getTxHash());
            }

            synchronized (condVar) {
                condVar.wait(5000);
                assertEquals(expectedEventNum[i], eventList.size());
            }
            em.stop();
            LOG.infoExiting("close");

            try {
                em.stop();
            }
            catch(IllegalStateException ex) {
                LOG.info(ex.getMessage());
            }
        }
    }
}
